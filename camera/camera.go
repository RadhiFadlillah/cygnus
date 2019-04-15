package camera

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	fp "path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

var developmentMode = false

// RaspiCam is controller for Raspberry Pi camera.
// It's used to capture the camera stream and process it.
type RaspiCam struct {
	DB *bolt.DB

	SaveToStorage bool
	StorageDir    string

	GenerateHlsSegments bool
	HlsSegmentsDir      string

	fps      int
	width    int
	height   int
	rotation int
	chStop   chan struct{}
}

// Start activates the camera, receive the stream and then process it
func (cam *RaspiCam) Start() error {
	logrus.Infoln("starting camera")

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
	cam.chStop = make(chan struct{})

	// Load settings from database
	cam.loadSetting()

	// Create cmd for child process
	cmdRaspivid := cam.genCmdRaspivid()
	cmdHlsSegments := cam.genCmdHlsSegments()
	cmdSaveToStorage := cam.genCmdSaveToStorage()

	// Create pipe for directing raspivid to save storage and HLS segments
	inHlsSegments, outHlsSegments := io.Pipe()
	inSaveToStorage, outSaveToStorage := io.Pipe()
	outRaspivid := io.MultiWriter(outHlsSegments, outSaveToStorage)

	cmdRaspivid.Stdout = outRaspivid
	cmdHlsSegments.Stdin = inHlsSegments
	cmdSaveToStorage.Stdin = inSaveToStorage

	defer func() {
		outHlsSegments.Close()
		outSaveToStorage.Close()
	}()

	// Run child process for processing the camera streams
	err = cmdRaspivid.Start()
	if err != nil {
		return fmt.Errorf("fail to start raspivid: %v", err)
	}
	logrus.Infoln("raspivid started")

	err = cmdHlsSegments.Start()
	if err != nil {
		return fmt.Errorf("fail to start HLS segmenter: %v", err)
	}
	logrus.Infoln("HLS segmenter started")

	err = cmdSaveToStorage.Start()
	if err != nil {
		return fmt.Errorf("fail to start video saver: %v", err)
	}
	logrus.Infoln("video saver started")

	// Block until stop request received
	select {
	case <-cam.chStop:
		cmdRaspivid.Process.Kill()
		cmdHlsSegments.Process.Kill()
		cmdSaveToStorage.Process.Kill()
	}

	logrus.Infoln("camera stopped")
	return nil
}

// Stop stops the camera streams.
func (cam *RaspiCam) Stop() {
	cam.chStop <- struct{}{}
}

func (cam *RaspiCam) loadSetting() {
	setting := make(map[string]string)
	cam.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("camera"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(key, val []byte) error {
			setting[string(key)] = string(val)
			return nil
		})

		return nil
	})

	fps, _ := strconv.Atoi(setting["fps"])
	rotation, _ := strconv.Atoi(setting["rotation"])
	resolutionParts := strings.SplitN(setting["resolution"], "x", 2)

	if fps <= 0 {
		fps = 30
	}

	switch rotation {
	case 0, 90, 180, 270:
	default:
		rotation = 0
	}

	width := 800
	height := 600
	if len(resolutionParts) == 2 {
		width, _ = strconv.Atoi(resolutionParts[0])
		height, _ = strconv.Atoi(resolutionParts[1])

		if width <= 0 || height <= 0 {
			width = 800
			height = 600
		}
	}

	cam.fps = fps
	cam.width = width
	cam.height = height
	cam.rotation = rotation
}

func (cam *RaspiCam) genCmdRaspivid() *exec.Cmd {
	if developmentMode {
		return exec.Command("nc", "-l", "-p", "5000")
	}

	cmdArgs := []string{
		"-t", "0",
		"-b", "0",
		"-qp", "30",
		"-ae", "16",
		"-a", "1036",
		"-a", "%Y-%m-%d %X",
		"-ex", "night",
		"-w", strconv.Itoa(cam.width),
		"-h", strconv.Itoa(cam.height),
		"-fps", strconv.Itoa(cam.fps),
		"-rot", strconv.Itoa(cam.rotation),
		"-vs", "-o", "-"}

	return exec.Command("raspivid", cmdArgs...)
}

func (cam *RaspiCam) genCmdSaveToStorage() *exec.Cmd {
	outputPath := fp.Join(cam.StorageDir, "%Y-%m-%d-%H:%M:%S.mp4")
	return exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-framerate", strconv.Itoa(cam.fps),
		"-i", "pipe:0",
		"-codec", "copy",
		"-f", "segment",
		"-strftime", "1",
		"-segment_time", "900",
		"-segment_format", "mp4",
		"-segment_format_options", "movflags=frag_keyframe+empty_moov",
		outputPath)
}

func (cam *RaspiCam) genCmdHlsSegments() *exec.Cmd {
	playlistPath := fp.Join(cam.HlsSegmentsDir, "playlist.m3u8")
	segmentPath := fp.Join(cam.HlsSegmentsDir, "%d.ts")
	return exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-framerate", strconv.Itoa(cam.fps),
		"-i", "pipe:0",
		"-codec", "copy",
		"-bsf", "h264_mp4toannexb",
		"-map", "0",
		"-hls_wrap", "10",
		"-hls_list_size", "10",
		"-hls_base_url", "/live/stream/",
		"-hls_segment_filename", segmentPath,
		"-hls_segment_type", "mpegts",
		"-hls_flags", "delete_segments+temp_file",
		playlistPath)
}
