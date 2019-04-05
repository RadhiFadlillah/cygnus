package handler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	fp "path/filepath"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// ServeLivePlaylist is handler for GET /live/playlist
// which serve HLS playlist for live stream
func (h *WebHandler) ServeLivePlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	playlistPath := fp.Join(h.HlsSegmentsDir, "playlist.m3u8")

	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, playlistPath)
}

// ServeLiveSegment is handler for GET /live/stream/:index
// which serve the HLS segment for live stream
func (h *WebHandler) ServeLiveSegment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	segmentIndex := ps.ByName("index")
	segmentPath := fp.Join(h.HlsSegmentsDir, segmentIndex)

	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	http.ServeFile(w, r, segmentPath)
}

// ServeVideoFile is handler for GET /video/:name.
// It serves the video file as it without any modifications.
func (h *WebHandler) ServeVideoFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	videoName := ps.ByName("name")
	videoPath := fp.Join(h.StorageDir, videoName+".mp4")

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, videoPath)
}

// ServeVideoPlaylist is handler for GET /video/:name/playlist
// which serve the HLS playlist for specified video
func (h *WebHandler) ServeVideoPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Get path to video file
	videoName := ps.ByName("name")
	videoPath := fp.Join(h.StorageDir, videoName+".mp4")

	// Run ffprobe
	cmd := exec.Command("ffprobe",
		"-loglevel", "fatal",
		"-print_format", "compact",
		"-show_entries", "format=duration",
		videoPath)

	output, err := cmd.CombinedOutput()
	checkError(err)

	// Parse video duration
	output = bytes.TrimSpace(output)
	outputParts := strings.SplitN(string(output), "=", 2)
	if len(outputParts) != 2 || outputParts[0] != "format|duration" {
		panic(fmt.Errorf("unable to parse video duration"))
	}

	vidDuration, err := strconv.ParseFloat(outputParts[1], 64)
	checkError(err)

	// Create playlist file
	buffer := new(bytes.Buffer)
	fmt.Fprintln(buffer, "#EXTM3U")
	fmt.Fprintln(buffer, "#EXT-X-VERSION:3")
	fmt.Fprintln(buffer, "#EXT-X-PLAYLIST-TYPE:VOD")
	fmt.Fprintln(buffer, "#EXT-X-TARGETDURATION:30")
	fmt.Fprintln(buffer, "#EXT-X-MEDIA-SEQUENCE:0")
	fmt.Fprintln(buffer, "#EXT-X-ALLOW-CACHE:YES")

	segmentIndex := 0
	for leftover := vidDuration; leftover > 0; leftover -= 30.0 {
		segmentLength := float64(30.0)
		if leftover < segmentLength {
			segmentLength = leftover
		}

		fmt.Fprintf(buffer, "#EXTINF:%f,\n", segmentLength)
		fmt.Fprintf(buffer, "/video/%s/stream/%d.ts\n", videoName, segmentIndex)
		segmentIndex++
	}

	fmt.Fprintln(buffer, "#EXT-X-ENDLIST")

	// Serve playlist
	w.Header().Set("Content-Type", "application/x-mpegURL")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	io.Copy(w, buffer)
}

// ServeVideoSegment is handler for GET /video/:name/stream/:index
// which serve the HLS segment for specified video
func (h *WebHandler) ServeVideoSegment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Get path to video file
	videoName := ps.ByName("name")
	videoPath := fp.Join(h.StorageDir, videoName+".mp4")

	// Calculate start time
	strIndex := ps.ByName("index")
	strIndex = strings.TrimSuffix(strIndex, fp.Ext(strIndex))
	index, err := strconv.Atoi(strIndex)
	checkError(err)

	startTime := float64(index * 30)
	if startTime < 0 {
		startTime = 0
	}

	// Cut video using ffmpeg
	buffer := new(bytes.Buffer)
	cmd := exec.Command("ffmpeg",
		"-loglevel", "fatal",
		"-ss", fmt.Sprintf("%f", startTime),
		"-i", videoPath,
		"-t", "30.0",
		"-codec", "copy",
		"-bsf", "h264_mp4toannexb",
		"-map", "0",
		"-f", "segment",
		"-segment_time", "30.0",
		"-segment_format", "mpegts",
		"-force_key_frames", "expr:gte(t,n_forced*30.000)",
		"-initial_offset", fmt.Sprintf("%f", startTime),
		"pipe:out%d.ts")
	cmd.Stdout = buffer

	err = cmd.Run()
	checkError(err)

	// Serve segment
	w.Header().Set("Content-Type", "video/MP2T")
	w.Header().Set("Cache-Control", "max-age=3600")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	io.Copy(w, buffer)
}
