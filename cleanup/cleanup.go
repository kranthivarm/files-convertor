package cleanup

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

const (
	MaxAge      = 1 * time.Hour
	SweepEvery  = 30 * time.Minute
)

func Start(dirs ...string) {
	go func() {
		sweep(dirs)
		ticker := time.NewTicker(SweepEvery)
		defer ticker.Stop()
		for range ticker.C {
			sweep(dirs)
		}
	}()
	log.Printf("Cleanup: files older than %v will be removed every %v", MaxAge, SweepEvery)
}

func sweep(dirs []string) {
	cutoff := time.Now().Add(-MaxAge)
	total := 0

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue 
		}
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				if e.IsDir() {
					if err := os.RemoveAll(path); err == nil {
						total++
					}
				} else {
					if err := os.Remove(path); err == nil {
						total++
					}
				}
			}
		}
	}

	if total > 0 {
		log.Printf("Cleanup: removed %d old file(s)/dir(s)", total)
	}
}