package inmem

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type (
	// Storage struct
	Storage struct {
		sync.RWMutex
		kv map[string]value
	}

	value struct {
		v   []byte
		exp time.Time
	}
)

// New inmemory storage
func New() *Storage {
	return &Storage{
		kv: make(map[string]value),
	}
}

// Set key-value in storage
func (s *Storage) Set(_ context.Context, key string, v []byte, ttl time.Duration) error {
	s.Lock()
	defer s.Unlock()

	if val, ok := s.kv[key]; !ok || time.Now().After(val.exp) {
		s.kv[key] = value{v: v, exp: time.Now().Add(ttl)}
		return nil
	}

	return fmt.Errorf("key %s already exists", key)
}

// Get value by key
func (s *Storage) Get(_ context.Context, key string) ([]byte, error) {
	s.RLock()
	defer s.RUnlock()

	if v, ok := s.kv[key]; ok {
		return v.v, nil
	}

	return nil, fmt.Errorf("could not found record with key=%s", key)
}

// Delete value by key
func (s *Storage) Delete(_ context.Context, key string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.kv[key]; !ok {
		return fmt.Errorf("could not found record with key=%s", key)
	}

	delete(s.kv, key)
	return nil
}
