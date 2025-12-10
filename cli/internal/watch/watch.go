// Package watch provides file watching functionality for schema changes.
package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a file for changes
type Watcher struct {
	file     string
	callback func() error
	watcher  *fsnotify.Watcher
	done     chan bool
}

// NewWatcher creates a new file watcher
func NewWatcher(file string, callback func() error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}

	absPath, err := filepath.Abs(file)
	if err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Watch the directory containing the file
	dir := filepath.Dir(absPath)
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to watch directory: %w", err)
	}

	return &Watcher{
		file:     absPath,
		callback: callback,
		watcher:  watcher,
		done:     make(chan bool),
	}, nil
}

// Start starts watching the file
func (w *Watcher) Start() error {
	// Initial callback if needed
	if err := w.callback(); err != nil {
		return fmt.Errorf("initial callback failed: %w", err)
	}

	go func() {
		debounceTimer := time.NewTimer(500 * time.Millisecond)
		debounceTimer.Stop()
		var debounceCh <-chan time.Time

		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}

				// Check if the watched file was modified
				if event.Op&fsnotify.Write == fsnotify.Write {
					eventPath, err := filepath.Abs(event.Name)
					if err == nil && eventPath == w.file {
						// Debounce: reset timer on each event
						debounceTimer.Reset(500 * time.Millisecond)
						debounceCh = debounceTimer.C
					}
				}

			case <-debounceCh:
				// File was modified, trigger callback
				if err := w.callback(); err != nil {
					fmt.Fprintf(os.Stderr, "Watch callback error: %v\n", err)
				}
				debounceCh = nil

			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				fmt.Fprintf(os.Stderr, "Watch error: %v\n", err)

			case <-w.done:
				return
			}
		}
	}()

	return nil
}

// Stop stops watching the file
func (w *Watcher) Stop() error {
	close(w.done)
	return w.watcher.Close()
}
