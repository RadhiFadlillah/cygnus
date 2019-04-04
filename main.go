//go:generate go run assets-generator.go

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	fp "path/filepath"
	"time"

	"github.com/shirou/gopsutil/disk"

	"github.com/RadhiFadlillah/cygnus/camera"
	"github.com/RadhiFadlillah/cygnus/handler"
	"github.com/julienschmidt/httprouter"
	cch "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var (
	portNumber     = 8080
	maxStorageSize = uint64(1024)

	camWidth  = 800
	camHeight = 600
	camFlip   = false

	dbPath      = "cygnus.db"
	storageDir  = "temp/storage"
	segmentsDir = "temp/segments"
)

func main() {
	// Make sure required directories exists
	err := os.MkdirAll(fp.Dir(dbPath), os.ModePerm)
	if err != nil {
		logrus.Fatalln("failed to create database dir:", err)
	}

	err = os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		logrus.Fatalln("failed to create storage dir:", err)
	}

	err = os.MkdirAll(segmentsDir, os.ModePerm)
	if err != nil {
		logrus.Fatalln("failed to create live segments dir:", err)
	}

	// Open database
	db, err := bolt.Open(dbPath, os.ModePerm, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	// Prepare channel
	chError := make(chan error)
	defer close(chError)

	// Start camera, web server and storage cleaner
	go startCamera(chError)
	go serveWebView(db, chError)
	go cleanStorage(chError)

	// Watch channel until error received
	select {
	case err := <-chError:
		logrus.Fatalln(err)
	}
}

func startCamera(chError chan error) {
	cam := camera.RaspiCam{
		Width:    camWidth,
		Height:   camHeight,
		FlipView: camFlip,

		GenerateHlsSegments: true,
		HlsSegmentsDir:      segmentsDir,

		SaveToStorage: true,
		StorageDir:    storageDir,
	}

	err := cam.Start()
	if err != nil {
		chError <- fmt.Errorf("camera error: %v", err)
	}
}

func serveWebView(db *bolt.DB, chError chan error) {
	hdl := handler.WebHandler{
		DB:             db,
		StorageDir:     storageDir,
		HlsSegmentsDir: segmentsDir,
		UserCache:      cch.New(time.Hour, 10*time.Minute),
		SessionCache:   cch.New(time.Hour, 10*time.Minute),
	}

	hdl.PrepareLoginCache()

	// Create router
	router := httprouter.New()

	// Serve files
	router.GET("/fonts/*filepath", hdl.ServeFile)
	router.GET("/res/*filepath", hdl.ServeFile)
	router.GET("/css/*filepath", hdl.ServeFile)
	router.GET("/js/*filepath", hdl.ServeJsFile)

	// Serve UI
	router.GET("/", hdl.ServeIndexPage)
	router.GET("/login", hdl.ServeLoginPage)
	router.GET("/live/playlist", hdl.ServeLivePlaylist)
	router.GET("/live/stream/:index", hdl.ServeLiveSegment)
	router.GET("/video/:name", hdl.ServeVideoFile)
	router.GET("/video/:name/playlist", hdl.ServeVideoPlaylist)
	router.GET("/video/:name/stream/:index", hdl.ServeVideoSegment)

	// Serve API
	router.POST("/api/login", hdl.APILogin)
	router.POST("/api/logout", hdl.APILogout)
	router.GET("/api/storage", hdl.APIGetStorageFiles)
	router.GET("/api/user", hdl.APIGetUsers)
	router.POST("/api/user", hdl.APIInsertUser)
	router.DELETE("/api/user/:username", hdl.APIDeleteUser)

	// Panic handler
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, arg interface{}) {
		http.Error(w, fmt.Sprint(arg), 500)
	}

	// Serve app
	strPortNumber := fmt.Sprintf(":%d", portNumber)
	logrus.Println("web server running in port " + strPortNumber)
	err := http.ListenAndServe(strPortNumber, router)
	if err != nil {
		chError <- fmt.Errorf("web server error: %v", err)
	}
}

// cleanStorage removes old video from storage when :
// - there are too many vids, which combined size > maxStorageSize
// - free space is less than 500MB
//
// Unlike startCamera and serveWebView, it's fine if removal failed.
// So, if error happened, we just add warning log.
func cleanStorage(chError chan error) {
	logWarn := func(a ...interface{}) {
		logrus.Warnln(a...)
	}

	for {
		time.Sleep(15 * time.Minute)

		// Get storage dir stat
		stat, err := disk.Usage(storageDir)
		if err != nil {
			logWarn("clean storage error: get stat failed:", err)
			continue
		}

		// If usage is more than max size, or free storage is less than 500 MB,
		// remove old recorded video.
		if (maxStorageSize > 0 && stat.Used > maxStorageSize) || stat.Free < 500*1000^2 {
			dirItems, err := ioutil.ReadDir(storageDir)
			if err != nil {
				logWarn("clean storage error: get items failed:", err)
				continue
			}

			oldestVideo := ""
			for _, item := range dirItems {
				if !item.IsDir() && fp.Ext(item.Name()) == ".mp4" {
					oldestVideo = item.Name()
					break
				}
			}

			err = os.Remove(fp.Join(storageDir, oldestVideo))
			if err != nil {
				logWarn("clean storage error: remove failed:", err)
				continue
			}

			logrus.Println("removing old video:", oldestVideo)
		}
	}
}
