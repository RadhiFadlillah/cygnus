package camera

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	fp "path/filepath"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
)

var developmentMode = false

// RaspiCam is controller for Raspberry Pi camera.
// It's used to capture the camera stream and process it.
type RaspiCam struct {
	Width    int
	Height   int
	FlipView bool

	SaveToStorage bool
	StorageDir    string

	GenerateHlsSegments bool
	HlsSegmentsDir      string

	waitGroup   sync.WaitGroup
	chError     chan error
	chStop      chan struct{}
	chChildStop map[string]chan struct{}
}

// Start activates the camera, receive the stream and then process it
func (cam *RaspiCam) Start() error {
	logrus.Infoln("starting camera")

	// Make sure needed directories exists
	err := os.MkdirAll(cam.StorageDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create save dir %s: %v", cam.StorageDir, err)
	}

	err = os.MkdirAll(cam.HlsSegmentsDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create live segments dir %s: %v", cam.HlsSegmentsDir, err)
	}

	// If the HLS segments dir is not empty, remove its contents
	dirItems, err := ioutil.ReadDir(cam.HlsSegmentsDir)
	if err != nil {
		return fmt.Errorf("failed to read live segments dir %s: %v", cam.HlsSegmentsDir, err)
	}

	for _, item := range dirItems {
		itemPath := fp.Join(cam.HlsSegmentsDir, item.Name())
		if item.IsDir() {
			os.RemoveAll(itemPath)
		} else {
			os.Remove(itemPath)
		}
	}

	// Create channels
	cam.waitGroup = sync.WaitGroup{}
	cam.chError = make(chan error, 3)
	cam.chStop = make(chan struct{})
	cam.chChildStop = map[string]chan struct{}{
		"raspivid":      make(chan struct{}),
		"hlsSegment":    make(chan struct{}),
		"saveToStorage": make(chan struct{}),
	}

	// Prepare pipes for future use.
	inputHlsSegments, outputHlsSegments := io.Pipe()
	inputSaveToStorage, outputSaveToStorage := io.Pipe()
	pipeWriters := io.MultiWriter(outputHlsSegments, outputSaveToStorage)

	// Run children process for processing the camera streams
	go cam.startRaspivid(pipeWriters)
	go cam.saveToStorage(inputSaveToStorage)
	go cam.generateHlsSegments(inputHlsSegments)

	// Block until error or stop request received
	select {
	case err = <-cam.chError:
	case <-cam.chStop:
	}

	// Stop all children and close its input
	for name := range cam.chChildStop {
		close(cam.chChildStop[name])
	}

	outputHlsSegments.Close()
	outputSaveToStorage.Close()

	// Once all children stopped, close channels
	cam.waitGroup.Wait()
	close(cam.chError)
	close(cam.chStop)

	logrus.Infoln("camera stopped")
	return err
}

// Stop stops the camera streams.
func (cam *RaspiCam) Stop() {
	logrus.Infoln("stopping camera")
	cam.chStop <- struct{}{}
}

func (cam *RaspiCam) startRaspivid(output io.Writer) {
	logrus.Infoln("starting raspivid")

	// Register wait group
	cam.waitGroup.Add(1)
	defer cam.waitGroup.Done()

	// Prepare command for raspivid.
	// If we are in development mode, we will work from our workstation.
	// So, instead of using raspivid, we will use netcat to receive stream from Raspberry Pi.
	var cmd *exec.Cmd
	if developmentMode {
		cmd = exec.Command("nc", "-l", "-p", "5000")
	} else {
		cmdArgs := []string{
			"-t", "0",
			"-b", "0",
			"-qp", "30",
			"-fps", "30",
			"-ae", "16",
			"-a", "1036",
			"-a", "%Y-%m-%d %X",
			"-w", strconv.Itoa(cam.Width),
			"-h", strconv.Itoa(cam.Height),
			"-vs", "-o", "-"}

		if cam.FlipView {
			cmdArgs = append(cmdArgs, "-hf", "-vf")
		}

		cmd = exec.Command("raspivid", cmdArgs...)
	}

	// Start raspivid
	cmd.Stdout = output
	err := cmd.Start()
	if err != nil {
		cam.chError <- err
		return
	}

	// Watch for stop request
	go func() {
		select {
		case <-cam.chChildStop["raspivid"]:
			cmd.Process.Kill()
		}
	}()

	// Wait until cmd stopped
	cam.chError <- cmd.Wait()
	logrus.Infoln("raspivid stopped")
}

func (cam *RaspiCam) saveToStorage(input io.Reader) {
	if !cam.SaveToStorage {
		return
	}

	logrus.Infoln("starting ffmpeg for save to storage")

	// Register wait group
	cam.waitGroup.Add(1)
	defer cam.waitGroup.Done()

	// Prepare ffmpeg for saving video's segments
	outputPath := fp.Join(cam.StorageDir, "%Y-%m-%d-%H:%M:%S.mp4")
	cmd := exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-framerate", "30",
		"-i", "pipe:0",
		"-codec", "copy",
		"-f", "segment",
		"-strftime", "1",
		"-segment_time", "900",
		"-segment_format", "mp4",
		"-segment_format_options", "movflags=frag_keyframe+empty_moov",
		outputPath)
	cmd.Stdin = input

	// Start ffmpeg
	err := cmd.Start()
	if err != nil {
		cam.chError <- err
		return
	}

	// Watch for stop request
	go func() {
		select {
		case <-cam.chChildStop["saveToStorage"]:
			cmd.Process.Kill()
		}
	}()

	// Wait until cmd stopped
	cam.chError <- cmd.Wait()
	logrus.Infoln("ffmpeg for save to storage stopped")
}

func (cam *RaspiCam) generateHlsSegments(input io.Reader) {
	if !cam.GenerateHlsSegments {
		return
	}

	logrus.Infoln("starting ffmpeg for HLS segments")

	// Register wait group
	cam.waitGroup.Add(1)
	defer cam.waitGroup.Done()

	// Prepare ffmpeg for generating HLS segments
	playlistPath := fp.Join(cam.HlsSegmentsDir, "playlist.m3u8")
	segmentPath := fp.Join(cam.HlsSegmentsDir, "%d.ts")
	cmd := exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-framerate", "30",
		"-i", "pipe:0",
		"-codec", "copy",
		"-bsf", "h264_mp4toannexb",
		"-map", "0",
		"-hls_base_url", "/stream/live/",
		"-hls_segment_filename", segmentPath,
		"-hls_segment_type", "mpegts",
		"-hls_flags", "delete_segments+temp_file",
		playlistPath)
	cmd.Stdin = input

	// Start ffmpeg
	err := cmd.Start()
	if err != nil {
		cam.chError <- err
		return
	}

	// Watch for stop request
	go func() {
		select {
		case <-cam.chChildStop["hlsSegment"]:
			cmd.Process.Kill()
		}
	}()

	// Wait until cmd stopped
	cam.chError <- cmd.Wait()
	logrus.Infoln("ffmpeg for HLS segments stopped")
}
