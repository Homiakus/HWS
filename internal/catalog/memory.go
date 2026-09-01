package catalog

import (
	"fmt"
	"sync"

	"github.com/Homiakus/HWS/internal/domain"
)

type Memory struct {
	mu    sync.RWMutex
	items map[string]domain.DesiredState
}

func NewMemory() *Memory {
	return &Memory{items: make(map[string]domain.DesiredState)}
}

func key(id domain.WorkspaceID, revision string) string {
	return string(id) + "@" + revision
}

func (m *Memory) Put(desired domain.DesiredState) error {
	if err := desired.Validate(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items[key(desired.WorkspaceID, desired.Revision)] = desired
	return nil
}

func (m *Memory) Resolve(id domain.WorkspaceID, revision string) (domain.DesiredState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	desired, ok := m.items[key(id, revision)]
	if !ok {
		return domain.DesiredState{}, fmt.Errorf("catalog: workspace %s revision %s not found", id, revision)
	}
	return desired, nil
}
