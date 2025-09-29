package main

import "sync"

type Store struct {
	mu sync.RWMutex
	m  map[string]Client // mac -> Client
}

func NewStore() *Store { return &Store{m: make(map[string]Client)} }

func (s *Store) Upsert(c Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c.LastSeen = c.LastSeen.UTC()
	s.m[c.Mac] = c
}

func (s *Store) Get(mac string) (Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.m[mac]
	return c, ok
}

func (s *Store) List() []Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Client, 0, len(s.m))
	for _, v := range s.m {
		out = append(out, v)
	}
	return out
}

func (s *Store) Delete(mac string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, mac)
}
