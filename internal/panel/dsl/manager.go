package dsl

import (
	"sync"

	"github.com/Homiakus/HWS/internal/panel"
)

type Manager struct {
	mu       sync.RWMutex
	current  panel.Spec
	valid    bool
	revision uint64
	lastErr  error
}

func (m *Manager) Apply(src []byte) (panel.Spec, uint64, error) {
	s, err := Compile(src)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastErr = err
		return m.current, m.revision, err
	}
	m.current = s
	m.valid = true
	m.revision++
	m.lastErr = nil
	return s, m.revision, nil
}
func (m *Manager) Current() (panel.Spec, uint64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current, m.revision, m.valid
}
func (m *Manager) LastError() error { m.mu.RLock(); defer m.mu.RUnlock(); return m.lastErr }
