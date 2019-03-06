package camera

import (
	"fmt"
	"io"
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

	GenerateHlsSegments bool
	HlsBaseURL          string
	HlsSegmentsDir      string
	HlsPlaylistPath     string

	SaveStreamToFile bool
	SaveDir          string

	waitGroup sync.WaitGroup
	chError   chan error
	chStop    chan bool
}

// Start activates the camera, receive the stream and then process it
func (cam *RaspiCam) Start() error {
	logrus.Infoln("starting camera")

	// Create channel
	cam.waitGroup = sync.WaitGroup{}
	cam.chError = make(chan error)
	cam.chStop = make(chan bool)

	// Prepare pipes for future use.
	pipeHlsSegmentsR, pipeHlsSegmentsW := io.Pipe()
	pipeSaveToFileR, pipeSaveToFileW := io.Pipe()
	pipeWriters := io.MultiWriter(pipeHlsSegmentsW, pipeSaveToFileW)

	// Start camera using raspivid.
	// However, if we are in development mode, we will work from our workstation.
	// So, instead of using raspivid, we will use netcat to receive stream from Raspberry Pi.
	var cmdRaspivid *exec.Cmd
	if developmentMode {
		cmdRaspivid = exec.Command("nc", "-l", "-p", "5000")
	} else {
		cmdArgs := []string{
			"-t", "0",
			"-b", "0",
			"-qp", "30",
			"-ae", "16",
			"-a", "1036",
			"-a", "%Y-%m-%d %X",
			"-w", strconv.Itoa(cam.Width),
			"-h", strconv.Itoa(cam.Height),
			"-vs", "-o", "-"}

		if cam.FlipView {
			cmdArgs = append(cmdArgs, "-hf", "-vf")
		}

		cmdRaspivid = exec.Command("raspivid", cmdArgs...)
	}

	cmdRaspivid.Stdout = pipeWriters

	err := cmdRaspivid.Start()
	if err != nil {
		return fmt.Errorf("failed to run raspivid: %v", err)
	}

	// Process the camera streams
	go cam.generateHlsSegments(pipeHlsSegmentsR)
	go cam.saveStreamToFile(pipeSaveToFileR)

	// Block until error or stop request received
	select {
	case err = <-cam.chError:
		cmdRaspivid.Process.Kill()
	case <-cam.chStop:
		cmdRaspivid.Process.Kill()
	}

	// Wait until all subprocess finished, then close channels
	cam.waitGroup.Wait()
	close(cam.chError)
	close(cam.chStop)
	return err
}

// Stop stops the camera streams.
func (cam *RaspiCam) Stop() {
	logrus.Infoln("stopping camera")
	cam.chStop <- true
}

func (cam *RaspiCam) generateHlsSegments(input io.Reader) {
	if !cam.GenerateHlsSegments {
		return
	}

	cam.waitGroup.Add(1)
	defer cam.waitGroup.Done()

	// Make sure directory exists
	err := os.MkdirAll(cam.HlsSegmentsDir, os.ModePerm)
	if err != nil {
		cam.chError <- err
		return
	}

	hlsPlaylistDir := fp.Dir(cam.HlsPlaylistPath)
	err = os.MkdirAll(hlsPlaylistDir, os.ModePerm)
	if err != nil {
		cam.chError <- err
		return
	}

	// Start generating segments
	segmentPath := fp.Join(cam.HlsSegmentsDir, "%d.ts")
	cmd := exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-i", "pipe:0",
		"-codec", "copy",
		"-bsf", "h264_mp4toannexb",
		"-map", "0",
		"-hls_time", "5",
		"-hls_list_size", "1",
		"-hls_base_url", "http://localhost:8080/stream/",
		"-hls_segment_filename", segmentPath,
		"-hls_segment_type", "mpegts",
		"-hls_flags", "delete_segments+temp_file",
		cam.HlsPlaylistPath)
	cmd.Stdin = input

	err = cmd.Run()
	if err != nil {
		cam.chError <- err
	}
}

func (cam *RaspiCam) saveStreamToFile(input io.Reader) {
	if !cam.SaveStreamToFile {
		return
	}

	cam.waitGroup.Add(1)
	defer cam.waitGroup.Done()

	// Make sure directory exists
	err := os.MkdirAll(cam.SaveDir, os.ModePerm)
	if err != nil {
		cam.chError <- err
		return
	}

	// Start saving streams
	outputPath := fp.Join(cam.SaveDir, "%Y-%m-%d-%H%M%S.mp4")
	cmd := exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-i", "pipe:0",
		"-codec", "copy",
		"-f", "segment",
		"-strftime", "1",
		"-segment_time", "900",
		"-segment_format", "mp4",
		"-segment_format_options", "movflags=frag_keyframe+empty_moov",
		outputPath)
	cmd.Stdin = input

	err = cmd.Run()
	if err != nil {
		cam.chError <- err
	}
}
