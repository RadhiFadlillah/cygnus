package watcher

import (
	"fmt"
	"os"
	fp "path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchFile watches change in the specified file.
func WatchFile(filePath string, eventHandler func(fsnotify.Event) error) error {
	// Make sure directory of the file is exists
	fileDir := fp.Dir(filePath)
	dirInfo, err := os.Stat(fileDir)
	if os.IsNotExist(err) || !dirInfo.IsDir() {
		return fmt.Errorf("directory %s does not exist", fileDir)
	}

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Add file directory to watcher
	err = watcher.Add(fileDir)
	if err != nil {
		return err
	}

	// Watch for file change
	lastEvent := struct {
		Name string
		Time time.Time
	}{}

	for {
		select {
		case event := <-watcher.Events:
			// Make sure the file is the one we want to watch
			if event.Name != filePath {
				continue
			}

			// In some OS, the write events fired twice.
			// To fix this, check if current event is happened less than one sec before.
			// If yes, skip this event.
			now := time.Now()
			eventName := fmt.Sprintf("%s: %s", event.Op.String(), filePath)
			if lastEvent.Name == eventName && now.Sub(lastEvent.Time).Seconds() <= 1.0 {
				continue
			}

			// Else, save this event and do something
			lastEvent = struct {
				Name string
				Time time.Time
			}{Name: eventName, Time: now}

			err = eventHandler(event)
			if err != nil {
				return fmt.Errorf("handler error: %v", err)
			}
		case err := <-watcher.Errors:
			if err != nil {
				return fmt.Errorf("watcher error: %v", err)
			}
		}
	}
}
