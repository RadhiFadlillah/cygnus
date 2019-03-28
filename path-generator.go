// +build !dev

package main

import (
	"os"
	fp "path/filepath"
)

func init() {
	homeDir := os.Getenv("HOME")
	cygnusDir := fp.Join(homeDir, "cygnus")

	if envDbPath, found := os.LookupEnv("ENV-CYGNUS-DB"); found {
		dbPath = envDbPath
	} else {
		dbPath = fp.Join(cygnusDir, "cygnus.db")
	}

	if envStorageDir, found := os.LookupEnv("ENV-CYGNUS-STORAGE"); found {
		storageDir = envStorageDir
	} else {
		storageDir = fp.Join(cygnusDir, "storage")
	}

	if envSegmentsDir, found := os.LookupEnv("ENV-CYGNUS-SEGMENTS"); found {
		segmentsDir = envSegmentsDir
	} else {
		segmentsDir = fp.Join(cygnusDir, "segments")
	}
}
