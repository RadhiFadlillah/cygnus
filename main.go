package main

import (
	"fmt"

	"github.com/RadhiFadlillah/solid-eye/camera"
	"github.com/sirupsen/logrus"
)

var (
	saveDir      = "temp/save"
	segmentsDir  = "temp/segments"
	playlistPath = "temp/playlist.m3u8"
)

func main() {
	// Prepare channel
	chError := make(chan error)
	defer close(chError)

	// Start camera
	startCamera(chError)

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
		HlsSegmentsDir:      segmentsDir,
		HlsPlaylistPath:     playlistPath,
		HlsBaseURL:          "http://localhost:8080/stream/",

		SaveStreamToFile: true,
		SaveDir:          saveDir,
	}

	go func() {
		err := cam.Start()
		if err != nil {
			chError <- fmt.Errorf("camera error: %v", err)
		}
	}()
}
