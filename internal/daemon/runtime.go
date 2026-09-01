package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Homiakus/HWS/internal/contexttree"
)

type Runtime struct {
	*Hub
	hierarchy *contexttree.Manager
	onTreeChanged func(uint64)
}

func NewRuntime(hub *Hub) *Runtime {
	if hub == nil {
		hub = NewHub(nil)
	}
	return &Runtime{Hub: hub, hierarchy: &contexttree.Manager{}}
}

func (r *Runtime) ConfigureHierarchy(path string) error {
	return r.hierarchy.Configure(path)
}

func (r *Runtime) SetTreeNotifier(notify func(uint64)) {
	r.onTreeChanged = notify
}

func (r *Runtime) TreeJSON() (string, error) {
	snapshot, ok := r.hierarchy.Snapshot()
	if !ok {
		return "", fmt.Errorf("hierarchy is unavailable")
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *Runtime) PathJSON(nodeID string) (string, error) {
	path, err := r.hierarchy.Path(nodeID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *Runtime) HealthJSON() (string, error) {
	base, err := r.Hub.HealthJSON()
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(base), &payload); err != nil {
		return "", err
	}
	payload["hierarchyRevision"] = r.hierarchy.Revision()
	payload["hierarchyValid"] = r.hierarchy.Valid()
	if lastErr := r.hierarchy.LastError(); lastErr != nil {
		payload["hierarchyError"] = lastErr.Error()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *Runtime) RunHierarchyMaintenance(ctx context.Context, interval time.Duration, report func(error)) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			changed, err := r.hierarchy.Poll()
			if err != nil {
				if report != nil {
					report(err)
				}
				continue
			}
			if changed && r.onTreeChanged != nil {
				r.onTreeChanged(r.hierarchy.Revision())
			}
		}
	}
}
