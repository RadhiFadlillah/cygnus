package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"

	"github.com/julienschmidt/httprouter"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

// APIGetSetting is handler for GET /api/setting
func (h *WebHandler) APIGetSetting(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Get list of usernames and setting
	users := h.getUsers()
	camera := h.getCameraSetting()

	// Decode to JSON
	data := map[string]interface{}{
		"users":  users,
		"camera": camera,
	}

	// Decode to JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&data)
	checkError(err)
}

// APIGetUsers is handler for GET /api/user
func (h *WebHandler) APIGetUsers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Get list of usernames from database
	users := h.getUsers()

	// Decode to JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&users)
	checkError(err)
}

// APIInsertUser is handler for POST /api/user
func (h *WebHandler) APIInsertUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Decode request
	var user User
	err = json.NewDecoder(r.Body).Decode(&user)
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

// APIDeleteUser is handler for DELETE /api/user/:username
func (h *WebHandler) APIDeleteUser(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

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

	// Delete user's sessions
	userSessions := []string{}
	if val, found := h.UserCache.Get(username); found {
		userSessions = val.([]string)
		for _, session := range userSessions {
			h.SessionCache.Delete(session)
		}

		h.UserCache.Delete(username)
	}

	fmt.Fprint(w, 1)
}

// APIGetCameraSetting is handler for GET /api/setting/camera
func (h *WebHandler) APIGetCameraSetting(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Get list of usernames from database
	setting := h.getCameraSetting()

	// Decode to JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(&setting)
	checkError(err)
}

// APISaveCameraSetting is handler for POST /api/setting/camera
func (h *WebHandler) APISaveCameraSetting(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Decode request
	setting := make(map[string]string)
	err = json.NewDecoder(r.Body).Decode(&setting)
	checkError(err)

	// Save setting to database
	h.DB.Update(func(tx *bolt.Tx) error {
		bucket, _ := tx.CreateBucketIfNotExists([]byte("camera"))
		bucket.Put([]byte("fps"), []byte(setting["fps"]))
		bucket.Put([]byte("rotation"), []byte(setting["rotation"]))
		bucket.Put([]byte("resolution"), []byte(setting["resolution"]))

		return nil
	})

	h.ChRestart <- true
	fmt.Fprint(w, 1)
}

// APIRebootCamera is handler for POST /api/setting/reboot
func (h *WebHandler) APIRebootCamera(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

	// Reboot system
	cmd := exec.Command("/bin/sh", "-c", "sudo reboot")
	bt, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("%v: %s", err, bt))
	}

	fmt.Fprint(w, 1)
}

func (h *WebHandler) getUsers() []string {
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

	return users
}

func (h *WebHandler) getCameraSetting() map[string]string {
	setting := make(map[string]string)
	h.DB.View(func(tx *bolt.Tx) error {
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

	return setting
}
