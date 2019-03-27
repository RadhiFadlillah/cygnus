package handler

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	fp "path/filepath"

	cch "github.com/patrickmn/go-cache"
	bolt "go.etcd.io/bbolt"
)

var developmentMode = false

// WebHandler is handler for serving the web interface.
type WebHandler struct {
	DB             *bolt.DB
	UserCache      *cch.Cache
	SessionCache   *cch.Cache
	StorageDir     string
	HlsSegmentsDir string
}

// PrepareCache prepares cache for future use
func (h *WebHandler) PrepareCache() {
	h.SessionCache.OnEvicted(func(key string, val interface{}) {
		username := val.(string)
		arr, found := h.UserCache.Get(username)
		if !found {
			return
		}

		sessionIDs := arr.([]string)
		for i := 0; i < len(sessionIDs); i++ {
			if sessionIDs[i] == key {
				sessionIDs = append(sessionIDs[:i], sessionIDs[i+1:]...)
				break
			}
		}

		h.UserCache.Set(username, sessionIDs, -1)
	})
}

func (h *WebHandler) validateSession(r *http.Request) error {
	// Get session-id from cookie
	sessionID, err := r.Cookie("session-id")
	if err != nil {
		if err == http.ErrNoCookie {
			return fmt.Errorf("session is not exist")
		}
		return err
	}

	// Make sure session is not expired yet
	if _, found := h.SessionCache.Get(sessionID.Value); !found {
		return fmt.Errorf("session has been expired")
	}

	return nil
}

func serveFile(w http.ResponseWriter, filePath string, cache bool) error {
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
