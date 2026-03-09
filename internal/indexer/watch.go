package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 50 * time.Millisecond
	maxDebounce   = 500 * time.Millisecond // flush after this even if events keep coming
)

// WatchEvent is sent on the Events channel after each successful update.
type WatchEvent struct {
	FilePath string
	Result   *UpdateResult
	Err      error
}

// Watcher watches a repository and incrementally updates the index.
type Watcher struct {
	root    string
	watcher *fsnotify.Watcher
	Events  chan WatchEvent
	done    chan struct{}
	once    sync.Once
}

// NewWatcher creates a watcher for the repository at root.
func NewWatcher(root string) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}

	w := &Watcher{
		root:    root,
		watcher: fw,
		Events:  make(chan WatchEvent, 32),
		done:    make(chan struct{}),
	}

	// Watch all subdirectories recursively
	if err := w.addDirs(root); err != nil {
		fw.Close()
		return nil, err
	}

	return w, nil
}

// Start begins watching. Blocks until Stop is called.
func (w *Watcher) Start() {
	// pending holds files waiting to be processed after debounce
	pending := make(map[string]time.Time) // filePath → first-seen time
	var mu sync.Mutex
	ticker := time.NewTicker(debounceDelay)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about supported source files
			if !isSupportedFile(event.Name) {
				continue
			}

			// Ignore chmod-only events
			if event.Op == fsnotify.Chmod {
				continue
			}

			mu.Lock()
			if _, exists := pending[event.Name]; !exists {
				// First time seeing this file — record when it entered the queue
				pending[event.Name] = time.Now()
			}
			mu.Unlock()

			// If a new directory was created, watch it
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.watcher.Add(event.Name)
				}
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.Events <- WatchEvent{Err: fmt.Errorf("watcher error: %w", err)}

		case <-ticker.C:
			mu.Lock()
			now := time.Now()
			var ready []string

			for path, firstSeen := range pending {
				// Process if:
				// - debounce window has passed (no new events for 50ms)
				// - OR max debounce reached (don't starve a busy file)
				if now.Sub(firstSeen) >= debounceDelay || now.Sub(firstSeen) >= maxDebounce {
					ready = append(ready, path)
					delete(pending, path)
				}
			}
			mu.Unlock()

			for _, path := range ready {
				w.processFile(path)
			}
		}
	}
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() {
	w.once.Do(func() {
		close(w.done)
		w.watcher.Close()
		close(w.Events)
	})
}

// processFile runs an incremental update for a single file.
func (w *Watcher) processFile(absPath string) {
	result, err := UpdateFile(w.root, absPath)
	w.Events <- WatchEvent{
		FilePath: absPath,
		Result:   result,
		Err:      err,
	}
}

// addDirs recursively adds all non-ignored directories to the watcher.
func (w *Watcher) addDirs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if skipWatchDirs[name] || strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		return w.watcher.Add(path)
	})
}

var skipWatchDirs = map[string]bool{
	".git":         true,
	".codeindex":   true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"__pycache__":  true,
}

func isSupportedFile(path string) bool {
	_, ok := detectLang(path)
	return ok
}
