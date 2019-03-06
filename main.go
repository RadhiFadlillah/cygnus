package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	fp "path/filepath"
	"strings"

	"github.com/RadhiFadlillah/solid-eye/camera"
	"github.com/RadhiFadlillah/solid-eye/watcher"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

var (
	saveDir      = "temp/save"
	segmentsDir  = "temp/segments"
	playlistPath = "temp/playlist.m3u8"
)

func main() {
	// Make sure all needed directories exist
	os.MkdirAll(saveDir, os.ModePerm)
	os.MkdirAll(segmentsDir, os.ModePerm)
	os.MkdirAll(fp.Dir(playlistPath), os.ModePerm)

	// Prepare channel
	chError := make(chan error)
	defer close(chError)

	// Start camera and watcher
	startCamera(chError)
	startPlaylistWatcher(chError)

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

func startPlaylistWatcher(chError chan error) {
	handler := func(event fsnotify.Event) error {
		if event.Op != fsnotify.Write {
			return nil
		}

		f, err := os.Open(playlistPath)
		if err != nil {
			return err
		}
		defer f.Close()

		buffer := bytes.NewBuffer(nil)
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#EXTINF") || strings.HasPrefix(line, "http") {
				buffer.WriteString(line)
			}
		}

		logrus.Println(buffer.String())
		return nil
	}

	go func() {
		err := watcher.WatchFile(playlistPath, handler)
		if err != nil {
			chError <- fmt.Errorf("playlist watcher error: %v", err)
		}
	}()
}
