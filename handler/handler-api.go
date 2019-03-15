package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"

	"github.com/julienschmidt/httprouter"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
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

// GetUsers is handler for GET /api/user
func (h *WebHandler) GetUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Get list of usernames from database
	users := []string{}
	h.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("user"))
		if bucket == nil {
			return nil
		}

		bucket.ForEach(func(key, val []byte) error {
			users = append(users, string(key))
			return nil
		})

		return nil
	})

	// Decode to JSON
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(&users)
	checkError(err)
}

// InsertUser is handler for POST /api/user
func (h *WebHandler) InsertUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Decode request
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	checkError(err)

	// Make sure that user not exists
	err = h.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("user"))
		if bucket == nil {
			return nil
		}

		if val := bucket.Get([]byte(user.Username)); val != nil {
			return fmt.Errorf("user %s already exists", user.Username)
		}

		return nil
	})
	checkError(err)

	// Hash password with bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	checkError(err)

	// Save user to database
	err = h.DB.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists([]byte("user"))
		return bucket.Put([]byte(user.Username), hashedPassword)
	})
	checkError(err)

	fmt.Fprint(w, 1)
}

// DeleteUser is handler for DELETE /api/user/:username
func (h *WebHandler) DeleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Get username
	username := ps.ByName("username")

	// Delete from database
	h.DB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("user"))
		if bucket == nil {
			return nil
		}

		bucket.Delete([]byte(username))
		return nil
	})

	fmt.Fprint(w, 1)
}
