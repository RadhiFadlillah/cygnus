//go:generate go run assets-generator.go

package main

import (
	"fmt"
	"net/http"
	fp "path/filepath"

	"github.com/RadhiFadlillah/cygnus/camera"
	"github.com/RadhiFadlillah/cygnus/handler"
	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

var (
	storageDir   = "temp/save"
	segmentsDir  = "temp/segments"
	playlistPath = "temp/playlist.m3u8"
)

func main() {
	// Prepare channel
	chError := make(chan error)
	defer close(chError)

	// Start camera and watcher
	startCamera(chError)
	serveWebView(chError)

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

func serveWebView(chError chan error) {
	hdl := handler.WebHandler{
		HlsSegmentsDir: segmentsDir,
	}

	router := httprouter.New()
	router.GET("/", hdl.ServeIndexPage)
	router.GET("/playlist/:name", hdl.ServeHlsPlaylist)
	router.GET("/stream/:name/:index", hdl.ServeHlsStream)

	go func() {
		logrus.Println("web server running in port :8080")
		err := http.ListenAndServe(":8080", router)
		if err != nil {
			chError <- fmt.Errorf("web server error: %v", err)
		}
	}()
}
