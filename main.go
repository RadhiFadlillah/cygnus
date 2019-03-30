//go:generate go run assets-generator.go

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	fp "path/filepath"
	"time"

	"github.com/RadhiFadlillah/cygnus/camera"
	"github.com/RadhiFadlillah/cygnus/handler"
	"github.com/julienschmidt/httprouter"
	cch "github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var (
	portNumber = 8080

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

	// Start camera and server
	startCamera(chError)
	serveWebView(db, chError)

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

	go func() {
		err := cam.Start()
		if err != nil {
			chError <- fmt.Errorf("camera error: %v", err)
		}
	}()
}

func serveWebView(db *bolt.DB, chError chan error) {
	hdl := handler.WebHandler{
		DB:             db,
		StorageDir:     storageDir,
		HlsSegmentsDir: segmentsDir,
		UserCache:      cch.New(time.Hour, 10*time.Minute),
		SessionCache:   cch.New(time.Hour, 10*time.Minute),
	}

	hdl.PrepareCache()

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

	go func() {
		strPortNumber := fmt.Sprintf(":%d", portNumber)
		logrus.Println("web server running in port " + strPortNumber)
		err := http.ListenAndServe(strPortNumber, router)
		if err != nil {
			chError <- fmt.Errorf("web server error: %v", err)
		}
	}()
}
