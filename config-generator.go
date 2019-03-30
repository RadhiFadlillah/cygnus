// +build !dev

package main

import (
	"os"
	fp "path/filepath"
	"strconv"
)

func init() {
	// Set server port number
	portNumber = 80
	if envPortNumber, found := os.LookupEnv("CYGNUS_PORT"); found {
		if intPort, err := strconv.Atoi(envPortNumber); intPort > 0 && err == nil {
			portNumber = intPort
		}
	}

	// Set camera config
	camWidth = 800
	if envCamWidth, found := os.LookupEnv("CYGNUS_CAM_WIDTH"); found {
		if intCamWidth, err := strconv.Atoi(envCamWidth); intCamWidth > 0 && err == nil {
			camWidth = intCamWidth
		}
	}

	camHeight = 600
	if envCamHeight, found := os.LookupEnv("CYGNUS_CAM_HEIGHT"); found {
		if intCamHeight, err := strconv.Atoi(envCamHeight); intCamHeight > 0 && err == nil {
			camHeight = intCamHeight
		}
	}

	_, camFlip = os.LookupEnv("CYGNUS_CAM_FLIP")

	// Set data directory
	homeDir := os.Getenv("HOME")
	cygnusDir := fp.Join(homeDir, "cygnus-data")

	dbPath = fp.Join(cygnusDir, "cygnus.db")
	if envDbPath, found := os.LookupEnv("CYGNUS_DB"); found {
		dbPath = envDbPath
	}

	storageDir = fp.Join(cygnusDir, "storage")
	if envStorageDir, found := os.LookupEnv("CYGNUS_STORAGE"); found {
		storageDir = envStorageDir
	}

	segmentsDir = fp.Join(cygnusDir, "segments")
	if envSegmentsDir, found := os.LookupEnv("CYGNUS_SEGMENTS"); found {
		segmentsDir = envSegmentsDir
	}
}
