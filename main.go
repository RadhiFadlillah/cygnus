//go:generate go run assets-generator.go

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	fp "path/filepath"

	"github.com/RadhiFadlillah/cygnus/camera"
	"github.com/RadhiFadlillah/cygnus/handler"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var (
	dbPath       = "cygnus.db"
	storageDir   = "temp/save"
	segmentsDir  = "temp/segments"
	playlistPath = "temp/playlist.m3u8"
)

func main() {
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
		Width:    800,
		Height:   600,
		FlipView: true,

		GenerateHlsSegments: true,
		HlsLiveSegmentsDir:  fp.Join(segmentsDir, "live"),

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
	}

	// Create router
	router := httprouter.New()

	// Serve files
	router.GET("/fonts/*filepath", hdl.ServeFile)
	router.GET("/res/*filepath", hdl.ServeFile)
	router.GET("/css/*filepath", hdl.ServeFile)
	router.GET("/js/*filepath", hdl.ServeJsFile)

	// Serve UI
	router.GET("/", hdl.ServeIndexPage)
	router.GET("/video/:name", hdl.ServeVideo)
	router.GET("/playlist/:name", hdl.ServeHlsPlaylist)
	router.GET("/stream/:name/:index", hdl.ServeHlsStream)

	// Serve API
	router.GET("/api/storage", hdl.GetStorageFiles)
	router.GET("/api/user", hdl.GetUsers)
	router.POST("/api/user", hdl.InsertUser)
	router.DELETE("/api/user/:username", hdl.DeleteUser)

	// Panic handler
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, arg interface{}) {
		http.Error(w, fmt.Sprint(arg), 500)
	}

	go func() {
		logrus.Println("web server running in port :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			chError <- fmt.Errorf("web server error: %v", err)
		}
	}()
}
