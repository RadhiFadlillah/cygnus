package handler

import (
	"net/http"
	"path"
	fp "path/filepath"
	"strings"

	"github.com/julienschmidt/httprouter"
)

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
