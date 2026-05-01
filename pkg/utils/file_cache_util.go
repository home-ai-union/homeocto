// Package utils provides utility functions for HomeClaw.
package utils

import (
	"fmt"
	"sync"
	"time"
)

// fileCacheManager manages FileCache instances with unified cleanup
type fileCacheManager struct {
	caches map[string]*FileCache
	mu     sync.RWMutex
	ticker *time.Ticker
	stopCh chan struct{}
}

// Global file cache manager instance
var defaultManager *fileCacheManager

// init initializes the global file cache manager with default cleanup interval
func init() {
	defaultManager = &fileCacheManager{
		caches: make(map[string]*FileCache),
		ticker: time.NewTicker(10 * time.Minute), // Default cleanup every 10 minutes
		stopCh: make(chan struct{}),
	}
	go defaultManager.cleanupLoop()
}

// cleanupLoop runs the periodic cleanup for all cached FileCache instances
func (m *fileCacheManager) cleanupLoop() {
	for {
		select {
		case <-m.ticker.C:
			m.cleanupAll()
		case <-m.stopCh:
			return
		}
	}
}

// cleanupAll runs Cleanup on all registered caches
func (m *fileCacheManager) cleanupAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, cache := range m.caches {
		if err := cache.Cleanup(); err != nil {
			// Log error but continue with other caches
			fmt.Printf("FileCache cleanup error for '%s': %v\n", name, err)
		}
	}
}

// Stop stops the cleanup timer
func (m *fileCacheManager) Stop() {
	m.ticker.Stop()
	close(m.stopCh)
}

// SetCleanupInterval changes the cleanup interval
func (m *fileCacheManager) SetCleanupInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ticker.Stop()
	m.ticker = time.NewTicker(interval)
}

// CreateFileCache creates or retrieves a cached FileCache instance.
// The name parameter is used to identify this cache instance.
// If a cache with the same name already exists, it returns the existing one.
func CreateFileCache(name string, config FileCacheConfig) (*FileCache, error) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	// Check if cache already exists
	if cache, exists := defaultManager.caches[name]; exists {
		return cache, nil
	}

	// Create new cache
	cache, err := newFileCache(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create FileCache '%s': %w", name, err)
	}

	// Cache the instance
	defaultManager.caches[name] = cache

	return cache, nil
}

// GetFileCache retrieves a cached FileCache instance by name
func GetFileCache(name string) (*FileCache, bool) {
	defaultManager.mu.RLock()
	defer defaultManager.mu.RUnlock()

	cache, exists := defaultManager.caches[name]
	return cache, exists
}

// RemoveFileCache removes a FileCache instance from the manager
func RemoveFileCache(name string) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	delete(defaultManager.caches, name)
}

// CleanupAll manually triggers cleanup for all cached FileCache instances
func CleanupAll() {
	defaultManager.cleanupAll()
}

// StopCleanupTimer stops the automatic cleanup timer
func StopCleanupTimer() {
	defaultManager.Stop()
}

// SetCleanupInterval changes the cleanup interval for all caches
func SetCleanupInterval(interval time.Duration) {
	defaultManager.SetCleanupInterval(interval)
}
