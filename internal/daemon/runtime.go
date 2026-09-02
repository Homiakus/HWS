package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Homiakus/HWS/internal/adapters/shelldesktop"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
	"github.com/Homiakus/HWS/internal/contexttree"
	"github.com/Homiakus/HWS/internal/domain"
	workspaceflow "github.com/Homiakus/HWS/internal/orchestration/workspace"
	"github.com/Homiakus/HWS/internal/shellaction"
)

type Runtime struct {
	*Hub
	hierarchy          *contexttree.Manager
	workspaces         *catalog.File
	shellActions       *shellaction.Broker
	workspaceLifecycle *workspaceflow.Lifecycle
	onTreeChanged      func(uint64)
}

func NewRuntime(hub *Hub) *Runtime {
	if hub == nil {
		hub = NewHub(nil)
	}
	return &Runtime{
		Hub:          hub,
		hierarchy:    &contexttree.Manager{},
		workspaces:   catalog.NewFile(),
		shellActions: shellaction.NewBroker(3 * time.Second),
	}
}

func (r *Runtime) ConfigureHierarchy(path string) error {
	return r.hierarchy.Configure(path)
}

func (r *Runtime) ConfigureWorkspaces(path string) error {
	return r.workspaces.Configure(path)
}

func (r *Runtime) OpenWorkspaceLifecycle(storePath string) error {
	if strings.TrimSpace(storePath) == "" {
		return errors.New("workspace lifecycle store path is required")
	}
	if r.workspaceLifecycle != nil {
		return errors.New("workspace lifecycle is already open")
	}
	if err := os.MkdirAll(filepath.Dir(storePath), 0o700); err != nil {
		return err
	}
	desktop := shelldesktop.New(r.Hub, r.shellActions)
	lifecycle, err := workspaceflow.OpenProduction(storePath, r.workspaces, reconcile.New(desktop))
	if err != nil {
		return err
	}
	r.workspaceLifecycle = lifecycle
	return nil
}

func (r *Runtime) Shutdown() error {
	if r.workspaceLifecycle == nil {
		return nil
	}
	lifecycle := r.workspaceLifecycle
	r.workspaceLifecycle = nil
	return lifecycle.Shutdown()
}

func (r *Runtime) SetShellActionEmitter(emit func(string)) {
	if emit == nil {
		r.shellActions.SetEmitter(nil)
		return
	}
	r.shellActions.SetEmitter(func(request shellaction.Request) error {
		payload, err := shellaction.EncodeRequest(request)
		if err != nil {
			return err
		}
		emit(payload)
		return nil
	})
}

func (r *Runtime) CompleteShellActionJSON(payload string) error {
	return r.shellActions.CompleteJSON(payload)
}

func (r *Runtime) ActivateWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return "", errors.New("workspace activation operation key is required")
	}
	desired, err := r.workspaces.Current(domain.WorkspaceID(strings.TrimSpace(workspaceID)))
	if err != nil {
		return "", err
	}
	return r.runWorkspaceMutation(desired.WorkspaceID, 20*time.Second, func(ctx context.Context) error {
		return r.workspaceLifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, operationKey)
	})
}

func (r *Runtime) RecoverWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, operationKey, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	return r.runWorkspaceMutation(id, 20*time.Second, func(ctx context.Context) error {
		return r.workspaceLifecycle.Recover(ctx, id, operationKey)
	})
}

func (r *Runtime) ResumeWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, operationKey, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	return r.runWorkspaceMutation(id, 20*time.Second, func(ctx context.Context) error {
		return r.workspaceLifecycle.Resume(ctx, id, operationKey)
	})
}

func (r *Runtime) SuspendWorkspaceJSON(workspaceID string) (string, error) {
	id, err := validateWorkspaceID(workspaceID)
	if err != nil {
		return "", err
	}
	return r.runWorkspaceMutation(id, 5*time.Second, func(ctx context.Context) error {
		return r.workspaceLifecycle.Suspend(ctx, id)
	})
}

func (r *Runtime) CloseWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, operationKey, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	return r.runWorkspaceMutation(id, 20*time.Second, func(ctx context.Context) error {
		return r.workspaceLifecycle.Close(ctx, id, operationKey)
	})
}

func (r *Runtime) WorkspaceStateJSON(workspaceID string) (string, error) {
	id, err := validateWorkspaceID(workspaceID)
	if err != nil {
		return "", err
	}
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return r.workspaceStateJSON(ctx, id)
}

func (r *Runtime) runWorkspaceMutation(workspaceID domain.WorkspaceID, timeout time.Duration, mutate func(context.Context) error) (string, error) {
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := mutate(ctx); err != nil {
		return "", err
	}
	return r.workspaceStateJSON(ctx, workspaceID)
}

func (r *Runtime) workspaceStateJSON(ctx context.Context, workspaceID domain.WorkspaceID) (string, error) {
	state, err := r.workspaceLifecycle.State(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func validateWorkspaceID(workspaceID string) (domain.WorkspaceID, error) {
	id := domain.WorkspaceID(strings.TrimSpace(workspaceID))
	if id == "" {
		return "", errors.New("workspace id is required")
	}
	return id, nil
}

func validateWorkspaceMutation(workspaceID, operationKey string) (domain.WorkspaceID, string, error) {
	id, err := validateWorkspaceID(workspaceID)
	if err != nil {
		return "", "", err
	}
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return "", "", errors.New("workspace operation key is required")
	}
	return id, operationKey, nil
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
	workspaceSnapshot := r.workspaces.Snapshot()
	payload["workspaceCatalogRevision"] = workspaceSnapshot.Revision
	payload["workspaceDefinitions"] = workspaceSnapshot.DefinitionCount
	payload["workspaceCatalogValid"] = r.workspaces.Valid()
	payload["workspaceLifecycleReady"] = r.workspaceLifecycle != nil
	if lastErr := r.workspaces.LastError(); lastErr != nil {
		payload["workspaceCatalogError"] = lastErr.Error()
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

func (r *Runtime) RunWorkspaceMaintenance(ctx context.Context, interval time.Duration, report func(error)) {
	r.workspaces.RunMaintenance(ctx, interval, report)
}
