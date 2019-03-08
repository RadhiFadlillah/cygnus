// +build dev

package handler

import (
	"net/http"
)

var assets = http.Dir("view")
