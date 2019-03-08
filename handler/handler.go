package handler

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	fp "path/filepath"

	"github.com/julienschmidt/httprouter"
)

// WebHandler is handler for serving the web interface.
type WebHandler struct {
	HlsSegmentsDir string
}

// ServeFile is handler for general file request
func (h *WebHandler) ServeFile(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
	http.ServeFile(w, r, playlistPath)
}

// ServeHlsStream is handler for GET /stream/:name/:index which serve the HLS segment for a video
func (h *WebHandler) ServeHlsStream(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	videoName := ps.ByName("name")
	segmentIndex := ps.ByName("index")
	segmentPath := fp.Join(h.HlsSegmentsDir, videoName, segmentIndex)

	w.Header().Set("Content-Type", "video/MP2T")
	http.ServeFile(w, r, segmentPath)
}

func serveFile(w http.ResponseWriter, filePath string) error {
	// Open file
	src, err := assets.Open(filePath)
	if err != nil {
		return err
	}
	defer src.Close()

	// Cache this file
	info, err := src.Stat()
	if err != nil {
		return err
	}

	etag := fmt.Sprintf(`W/"%x-%x"`, info.ModTime().Unix(), info.Size())
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "max-age=604800")

	// Set content type
	ext := fp.Ext(filePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}

	// Serve file
	_, err = io.Copy(w, src)
	return err
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
