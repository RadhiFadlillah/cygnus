package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
)

var rxSavedVideo = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(\d{2}:\d{2}:\d{2})\.mp4$`)

// GetStorageFiles is handler for GET /api/storage
func (h *WebHandler) GetStorageFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Read storage dir
	dirItems, err := ioutil.ReadDir(h.StorageDir)
	checkError(err)

	// Get list of day
	days := make(map[string][]string)
	for _, item := range dirItems {
		if !rxSavedVideo.MatchString(item.Name()) {
			continue
		}

		parts := rxSavedVideo.FindStringSubmatch(item.Name())
		day := parts[1]
		time := parts[2]

		days[day] = append(days[day], time)
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&days)
	checkError(err)
}
