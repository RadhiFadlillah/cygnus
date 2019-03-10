package handler

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	fp "path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var developmentMode = false

// WebHandler is handler for serving the web interface.
type WebHandler struct {
	HlsSegmentsDir string
}

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

func redirectPage(w http.ResponseWriter, r *http.Request, url string) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.Redirect(w, r, url, 301)
}

func fileExists(filePath string) bool {
	f, err := assets.Open(filePath)
	if f != nil {
		f.Close()
	}
	return err == nil || !os.IsNotExist(err)
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
