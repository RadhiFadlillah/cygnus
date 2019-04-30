//go:generate go run assets-generator.go

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	fp "path/filepath"
	"time"

	"github.com/RadhiFadlillah/cygnus/camera"
	"github.com/RadhiFadlillah/cygnus/handler"
	"github.com/julienschmidt/httprouter"
	cch "github.com/patrickmn/go-cache"
	"github.com/shirou/gopsutil/disk"
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
	db, err := prepareDatabase()
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	// Clean old videos in background
	go cleanStorage()

	// Prepare channels
	chError := make(chan error)
	chRestart := make(chan bool)
	defer func() {
		close(chError)
		close(chRestart)
	}()

	// Start CCTV system
	startCctvSystem(db, chError, chRestart)
}

func prepareDatabase() (*bolt.DB, error) {
	db, err := bolt.Open(dbPath, os.ModePerm, nil)
	if err != nil {
		return nil, err
	}

	db.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists([]byte("camera"))
		if bucket.Stats().KeyN == 0 {
			bucket.Put([]byte("fps"), []byte("30"))
			bucket.Put([]byte("rotation"), []byte("0"))
			bucket.Put([]byte("resolution"), []byte("800x600"))
		}

		return nil
	})

	return db, nil
}

func startCctvSystem(db *bolt.DB, chError chan error, chRestart chan bool) {
	// Prepare camera
	cam := &camera.RaspiCam{
		DB: db,

		GenerateHlsSegments: true,
		HlsSegmentsDir:      segmentsDir,

		SaveToStorage: true,
		StorageDir:    storageDir,
	}

	// Prepare web handler
	hdl := handler.WebHandler{
		DB:             db,
		StorageDir:     storageDir,
		HlsSegmentsDir: segmentsDir,
		UserCache:      cch.New(time.Hour, 10*time.Minute),
		SessionCache:   cch.New(time.Hour, 10*time.Minute),
		ChRestart:      chRestart,
	}

	hdl.PrepareLoginCache()

	// Prepare router
	router := httprouter.New()

	router.GET("/js/*filepath", hdl.ServeJsFile)
	router.GET("/res/*filepath", hdl.ServeFile)
	router.GET("/css/*filepath", hdl.ServeFile)
	router.GET("/fonts/*filepath", hdl.ServeFile)

	router.GET("/", hdl.ServeIndexPage)
	router.GET("/login", hdl.ServeLoginPage)
	router.GET("/live/playlist", hdl.ServeLivePlaylist)
	router.GET("/live/stream/:index", hdl.ServeLiveSegment)
	router.GET("/video/:name", hdl.ServeVideoFile)
	router.GET("/video/:name/playlist", hdl.ServeVideoPlaylist)
	router.GET("/video/:name/stream/:index", hdl.ServeVideoSegment)

	router.POST("/api/login", hdl.APILogin)
	router.POST("/api/logout", hdl.APILogout)
	router.GET("/api/storage", hdl.APIGetStorageFiles)

	router.GET("/api/user", hdl.APIGetUsers)
	router.POST("/api/user", hdl.APIInsertUser)
	router.DELETE("/api/user/:username", hdl.APIDeleteUser)

	router.GET("/api/setting", hdl.APIGetSetting)
	router.GET("/api/setting/camera", hdl.APIGetCameraSetting)
	router.POST("/api/setting/camera", hdl.APISaveCameraSetting)
	router.POST("/api/setting/reboot", hdl.APIRebootCamera)

	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, arg interface{}) {
		http.Error(w, fmt.Sprint(arg), 500)
	}

	// Prepare server
	serverAddr := fmt.Sprintf(":%d", portNumber)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Capture camera stream in background thread
	go func() {
		err := cam.Start()
		if err != nil {
			chError <- fmt.Errorf("camera error: %v", err)
		}
	}()

	// Serve web app in background thread
	go func() {
		logrus.Println("web server started in " + serverAddr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			chError <- fmt.Errorf("web server error: %v", err)
		}
	}()

	// Watch channel until error received, or restart request received
	select {
	case err := <-chError:
		logrus.Fatalln(err)
	case <-chRestart:
		logrus.Println("restart request received")

		cam.Stop()
		if err := server.Shutdown(context.Background()); err != nil {
			logrus.Fatalln("failed to shutdown server:", err)
		}
		logrus.Println("web server stopped")

		time.Sleep(3 * time.Second)
		startCctvSystem(db, chError, chRestart)
	}
}

// cleanStorage removes old video from storage when :
// - there are too many vids, which combined size > maxStorageSize
// - free space is less than 500MB
//
// Unlike startCamera and serveWebView, it's fine if removal failed.
// So, if error happened, we just add warning log.
func cleanStorage() {
	logWarn := func(a ...interface{}) {
		logrus.Warnln(a...)
	}

	for {
		// Get storage dir stat
		stat, err := disk.Usage(storageDir)
		if err != nil {
			logWarn("clean storage error: get stat failed:", err)
			continue
		}

		// If usage is more than max size, or free storage is less than 500 MB,
		// remove old recorded video.
		mbFree := float64(stat.Free) / 1000 / 1000
		if (maxStorageSize > 0 && stat.Used > maxStorageSize) || mbFree < 500 {
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

			logrus.Printf("free space %.0f MB, removing old video: %s", mbFree, oldestVideo)
		}

		time.Sleep(time.Minute)
	}
}
