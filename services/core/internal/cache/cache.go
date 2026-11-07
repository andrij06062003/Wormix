// Package cache replaces the Redis per-session scratch store of the C# server (Facades/Cache.cs) with an
// in-memory, concurrency-safe map: one JSON document per TCP session id, created at login and deleted when
// the session ends.
package cache

import "sync"

// Store maps session ids to their scratch JSON document.
type Store struct {
	mu   sync.Mutex
	data map[string]string
}

// New returns an empty store.
func New() *Store { return &Store{data: map[string]string{}} }

// Init resets the session's document to "{}" (Cache.InitCache).
func (s *Store) Init(sessionID string) {
	s.mu.Lock()
	s.data[sessionID] = "{}"
	s.mu.Unlock()
}

// Clear deletes the session's document (Cache.ClearCache).
func (s *Store) Clear(sessionID string) {
	s.mu.Lock()
	delete(s.data, sessionID)
	s.mu.Unlock()
}

// Get returns the session's document, or "{}" if there is none (Cache.GetCacheObject).
func (s *Store) Get(sessionID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.data[sessionID]; ok {
		return v
	}
	return "{}"
}

// Len is the number of stored documents.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.data)
}
