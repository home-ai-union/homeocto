// Package utils provides common utility functions.
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// GenerateUUID use nano
func GenerateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// BinaryCache stores resolved binary paths with sync.Once for thread-safe lazy initialization.
type BinaryCache struct {
	path string
	once sync.Once
}

// FindBinary locates an executable binary with caching support.
// Search order:
//  1. extraDirs (if provided)
//  2. Same directory as the current executable
//  3. Falls back to binaryName and relies on $PATH
//
// The cache parameter ensures the search is only performed once, subsequent calls
// return the cached result. Pass a pointer to a BinaryCache struct.
func FindBinary(binaryName string, cache *BinaryCache, extraDirs ...string) string {
	cache.once.Do(func() {
		if runtime.GOOS == "windows" {
			binaryName = binaryName + ".exe"
		}

		// Check extra directories first
		for _, dir := range extraDirs {
			if dir == "" {
				continue
			}
			candidate := filepath.Join(dir, binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				cache.path = candidate
				return
			}
		}

		// Check same directory as current executable
		if exe, err := os.Executable(); err == nil {
			candidate := filepath.Join(filepath.Dir(exe), binaryName)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				cache.path = candidate
				return
			}
		}

		cache.path = binaryName
	})
	return cache.path
}
