package handler

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	fp "path/filepath"
)

var developmentMode = false

// WebHandler is handler for serving the web interface.
type WebHandler struct {
	StorageDir     string
	HlsSegmentsDir string
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
