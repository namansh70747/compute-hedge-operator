package console

import "sync"

// Store keeps a bounded rolling history of float samples per key, for sparklines.
type Store struct {
	mu     sync.Mutex
	cap    int
	series map[string][]float64
}

// NewStore builds a store that retains at most capN samples per key.
func NewStore(capN int) *Store {
	return &Store{cap: capN, series: make(map[string][]float64)}
}

// Push appends a sample to a key, trimming to the retention cap.
func (s *Store) Push(key string, v float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	buf := append(s.series[key], v)
	if len(buf) > s.cap {
		buf = buf[len(buf)-s.cap:]
	}
	s.series[key] = buf
}

// Snapshot returns a copy of the samples for a key.
func (s *Store) Snapshot(key string) []float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.series[key]
	out := make([]float64, len(src))
	copy(out, src)
	return out
}
