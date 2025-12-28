package aic

import (
	"fmt"
	"strings"
	"sync"
)

// SequenceManager holds registered sequences.
type SequenceManager struct {
	mu  sync.RWMutex
	seq map[string]Sequence // key -> sequence
}

func NewSequenceManager() *SequenceManager {
	return &SequenceManager{
		seq: make(map[string]Sequence),
	}
}

func (m *SequenceManager) Register(s Sequence) error {
	if s == nil {
		return fmt.Errorf("sequence is nil")
	}
	k := strings.TrimSpace(s.Key())
	if len(k) != 1 {
		return fmt.Errorf("sequence key must be exactly 1 character, got %q", k)
	}
	k = strings.ToLower(k)

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.seq[k]; exists {
		return fmt.Errorf("sequence key %q already registered", k)
	}
	m.seq[k] = s
	return nil
}

func (m *SequenceManager) Get(key string) (Sequence, bool) {
	k := strings.ToLower(strings.TrimSpace(key))
	if len(k) != 1 {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.seq[k]
	return s, ok
}
