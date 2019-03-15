package handler

// User is person that given access to camera
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
