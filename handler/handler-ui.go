package handler

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	fp "path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

// ServeFile is handler for general file request
func (h *WebHandler) ServeFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := serveFile(w, r.URL.Path)
	checkError(err)
}

// ServeJsFile is handler for GET /js/
func (h *WebHandler) ServeJsFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	filePath := r.URL.Path
	fileName := path.Base(filePath)
	fileDir := path.Dir(filePath)

	if developmentMode && fp.Ext(fileName) == ".js" && strings.HasSuffix(fileName, ".min.js") {
		fileName = strings.TrimSuffix(fileName, ".min.js") + ".js"
		filePath = path.Join(fileDir, fileName)
		if fileExists(filePath) {
			redirectPage(w, r, filePath)
		}

		return
	}

	err := serveFile(w, r.URL.Path)
	checkError(err)
}

// ServeIndexPage is handler for GET /
func (h *WebHandler) ServeIndexPage(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := serveFile(w, "index.html")
	checkError(err)
}

// ServeHlsPlaylist is handler for GET /playlist/:name which serve the HLS playlist for a video
func (h *WebHandler) ServeHlsPlaylist(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	videoName := ps.ByName("name")
	playlistPath := fp.Join(h.HlsSegmentsDir, videoName, "playlist.m3u8")

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if videoName == "live" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	}

	http.ServeFile(w, r, playlistPath)
}

// ServeHlsStream is handler for GET /stream/:name/:index which serve the HLS segment for a video
func (h *WebHandler) ServeHlsStream(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	videoName := ps.ByName("name")
	segmentIndex := ps.ByName("index")
	segmentPath := fp.Join(h.HlsSegmentsDir, videoName, segmentIndex)

	w.Header().Set("Content-Type", "video/MP2T")
	if videoName == "live" {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	} else {
		w.Header().Set("Cache-Control", "max-age=1200")
	}

	http.ServeFile(w, r, segmentPath)
}

// ServeVideo is handler for GET /video/:name.
// If URL query download=true, it will send the file as is.
// Else, it will create HLS playlist which will be played by client.
func (h *WebHandler) ServeVideo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	videoName := ps.ByName("name")
	videoPath := fp.Join(h.StorageDir, videoName+".mp4")

	// Check if it's only for download
	if _, isDownload := r.URL.Query()["download"]; isDownload {
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeFile(w, r, videoPath)
		return
	}

	// Create directory for HLS file
	videoHlsDir := fp.Join(h.HlsSegmentsDir, videoName)
	err := os.MkdirAll(videoHlsDir, os.ModePerm)
	checkError(err)

	// Run ffmpeg for generating HLS segment
	playlistPath := fp.Join(videoHlsDir, "playlist.m3u8")
	segmentPath := fp.Join(videoHlsDir, "%d.ts")
	segmentURL := fmt.Sprintf("/stream/%s/", videoName)
	cmd := exec.Command("ffmpeg", "-y",
		"-loglevel", "fatal",
		"-i", videoPath,
		"-codec", "copy",
		"-bsf", "h264_mp4toannexb",
		"-map", "0",
		"-hls_time", "30",
		"-hls_list_size", "0",
		"-hls_base_url", segmentURL,
		"-hls_segment_filename", segmentPath,
		"-hls_segment_type", "mpegts",
		playlistPath)

	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("%v: %s", err, out))
	}

	// Redirect to serving HLS stream
	playlistURL := path.Join("/", "playlist", videoName)
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.Redirect(w, r, playlistURL, 301)
}
