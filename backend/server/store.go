package server

import "sync"

// Store keeps each table's state, saved after every change, so tables can be
// brought back after a restart.
type Store interface {
	Save(name string, state []byte) error
	Delete(name string) error
	Load() (map[string][]byte, error) // every saved table, by name
}

// memStore keeps tables in memory, so they don't survive a restart.
type memStore struct {
	mu     sync.Mutex
	tables map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{tables: map[string][]byte{}}
}

func (m *memStore) Save(name string, state []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tables[name] = state
	return nil
}

func (m *memStore) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tables, name)
	return nil
}

func (m *memStore) Load() (map[string][]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string][]byte, len(m.tables))
	for name, b := range m.tables {
		out[name] = b
	}
	return out, nil
}
