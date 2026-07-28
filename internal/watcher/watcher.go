package watcher

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

var watchedDirs = []string{"content", "templates", "static"}

const debounceDelay = 150 * time.Millisecond

func Watch(onChange func(string)) (func() error, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	for _, dir := range watchedDirs {
		if err := filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return w.Add(path)
			}
			return nil
		}); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			w.Close()
			return nil, fmt.Errorf("watch %s: %w", dir, err)
		}
	}

	changeCh := make(chan string, 1)
	go debounceChanges(changeCh, onChange)

	go func() {
		defer close(changeCh)

		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						_ = w.Add(event.Name)
					}
				}
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
					select {
					case changeCh <- event.Name:
					default:
						<-changeCh
						changeCh <- event.Name
					}
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				fmt.Println("Watcher error:", err)
			}
		}
	}()

	return w.Close, nil
}

func debounceChanges(changeCh <-chan string, onChange func(string)) {
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
		latest  string
	)

	for {
		select {
		case name, ok := <-changeCh:
			if !ok {
				if timer != nil {
					timer.Stop()
				}
				return
			}
			latest = name
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(debounceDelay)
			timerCh = timer.C
		case <-timerCh:
			onChange(latest)
			timerCh = nil
		}
	}
}
