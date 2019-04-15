package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	"github.com/gofrs/uuid"
	"github.com/julienschmidt/httprouter"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
)

var rxSavedVideo = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-(\d{2}:\d{2}:\d{2})\.mp4$`)

// APILogin is handler for POST /api/login
func (h *WebHandler) APILogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Decode request
	var request LoginRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	checkError(err)

	// Prepare function to generate session
	genSession := func(expTime time.Duration) {
		// Create session ID
		sessionID, err := uuid.NewV4()
		checkError(err)

		// Save session ID to cache
		strSessionID := sessionID.String()
		h.SessionCache.Set(strSessionID, request.Username, expTime)

		// Save user's session IDs to cache as well
		// useful for mass logout
		sessionIDs := []string{strSessionID}
		if val, found := h.UserCache.Get(request.Username); found {
			sessionIDs = val.([]string)
			sessionIDs = append(sessionIDs, strSessionID)
		}
		h.UserCache.Set(request.Username, sessionIDs, -1)

		// Return session ID to user in cookies
		http.SetCookie(w, &http.Cookie{
			Name:    "session-id",
			Value:   strSessionID,
			Path:    "/",
			Expires: time.Now().Add(expTime),
		})

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, strSessionID)
	}

	// Check if user's database is empty
	dbIsEmpty := false
	h.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("user"))
		dbIsEmpty = bucket == nil || bucket.Stats().KeyN == 0
		return nil
	})

	// If database still empty, and user uses default account, let him in
	if dbIsEmpty && request.Username == "admin" && request.Password == "admin" {
		genSession(time.Hour)
		return
	}

	// Get account data from database
	var hashedPassword []byte
	err = h.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("user"))
		if bucket == nil {
			return fmt.Errorf("user is not exist")
		}

		hashedPassword = bucket.Get([]byte(request.Username))
		if hashedPassword == nil {
			return fmt.Errorf("user is not exist")
		}

		return nil
	})
	checkError(err)

	// Compare password with database
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(request.Password))
	if err != nil {
		panic(fmt.Errorf("username and password don't match"))
	}

	// Calculate expiration time
	expTime := time.Hour
	if request.Remember > 0 {
		expTime = time.Duration(request.Remember) * time.Hour
	} else {
		expTime = -1
	}

	// Create session
	genSession(expTime)
}

// APILogout is handler for POST /api/logout
func (h *WebHandler) APILogout(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Get session ID
	sessionID, err := r.Cookie("session-id")
	if err != nil {
		if err == http.ErrNoCookie {
			panic(fmt.Errorf("session is expired"))
		} else {
			panic(err)
		}
	}

	h.SessionCache.Delete(sessionID.Value)
	fmt.Fprint(w, 1)
}

// APIGetStorageFiles is handler for GET /api/storage
func (h *WebHandler) APIGetStorageFiles(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Make sure session still valid
	err := h.validateSession(r)
	checkError(err)

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
