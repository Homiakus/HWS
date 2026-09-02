package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/adapters/shelldesktop"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
	"github.com/Homiakus/HWS/internal/contexttree"
	"github.com/Homiakus/HWS/internal/domain"
	workspaceflow "github.com/Homiakus/HWS/internal/orchestration/workspace"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
	"github.com/Homiakus/HWS/internal/shellaction"
)

const workspaceStatesSchemaVersion uint32 = 1

type WorkspaceStatesSnapshot struct {
	Schema          uint32                `json:"schema"`
	Revision        uint64                `json:"revision"`
	CatalogRevision uint64                `json:"catalogRevision"`
	States          []workspaceflow.State `json:"states"`
}

type Runtime struct {
	*Hub
	hierarchy          *contexttree.Manager
	workspaces         *catalog.File
	shellActions       *shellaction.Broker
	workspaceLifecycle *workspaceflow.Lifecycle
	onTreeChanged      func(uint64)

	workspaceStateMu       sync.RWMutex
	workspaceStateRevision uint64
	onWorkspaceChanged     func(string, uint64)

	shellSnapshotMu sync.RWMutex
	shellSnapshot   gnomeshell.Snapshot
}

func NewRuntime(hub *Hub) *Runtime {
	if hub == nil {
		hub = NewHub(nil)
	}
	return &Runtime{
		Hub:                    hub,
		hierarchy:              &contexttree.Manager{},
		workspaces:             catalog.NewFile(),
		shellActions:           shellaction.NewBroker(3 * time.Second),
		workspaceStateRevision: 1,
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
	desktop := shelldesktop.New(r, r.shellActions)
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

// ReplaceShellSnapshotJSON validates and retains the authoritative native
// topology/window geometry before publishing the same snapshot into the
// canonical surface aggregation path. Failed snapshots never replace the
// last-known-good placement observation.
func (r *Runtime) ReplaceShellSnapshotJSON(payload string) error {
	snapshot, err := gnomeshell.Decode([]byte(payload))
	if err != nil {
		return err
	}
	if err := r.Hub.ReplaceShellSnapshotJSON(payload); err != nil {
		return err
	}
	r.shellSnapshotMu.Lock()
	r.shellSnapshot = cloneShellSnapshot(snapshot)
	r.shellSnapshotMu.Unlock()
	return nil
}

func (r *Runtime) ShellSnapshot() gnomeshell.Snapshot {
	r.shellSnapshotMu.RLock()
	defer r.shellSnapshotMu.RUnlock()
	return cloneShellSnapshot(r.shellSnapshot)
}

func cloneShellSnapshot(in gnomeshell.Snapshot) gnomeshell.Snapshot {
	out := in
	out.Topology.Monitors = append([]gnomeshell.Monitor(nil), in.Topology.Monitors...)
	out.Apps = make([]gnomeshell.Application, len(in.Apps))
	for i := range in.Apps {
		out.Apps[i] = in.Apps[i]
		out.Apps[i].IdentityHints = append([]string(nil), in.Apps[i].IdentityHints...)
		out.Apps[i].Windows = append([]gnomeshell.Window(nil), in.Apps[i].Windows...)
	}
	return out
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
	state, err := r.workspaceState(ctx, id, "")
	if err != nil {
		return "", err
	}
	return marshalWorkspaceState(state)
}

func (r *Runtime) WorkspaceStatesJSON() (string, error) {
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	catalogSnapshot := r.workspaces.Snapshot()
	ids := make([]string, 0, len(catalogSnapshot.Active))
	for id := range catalogSnapshot.Active {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	states := make([]workspaceflow.State, 0, len(ids))
	for _, rawID := range ids {
		state, err := r.workspaceState(ctx, domain.WorkspaceID(rawID), catalogSnapshot.Active[rawID])
		if err != nil {
			return "", err
		}
		states = append(states, state)
	}

	snapshot := WorkspaceStatesSnapshot{
		Schema:          workspaceStatesSchemaVersion,
		Revision:        r.workspaceRevision(),
		CatalogRevision: catalogSnapshot.Revision,
		States:          states,
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(data), nil
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
	state, err := r.workspaceState(ctx, workspaceID, "")
	if err != nil {
		return "", err
	}
	value, err := marshalWorkspaceState(state)
	if err != nil {
		return "", err
	}
	r.notifyWorkspaceChanged(string(workspaceID))
	return value, nil
}

func (r *Runtime) workspaceState(ctx context.Context, workspaceID domain.WorkspaceID, definitionRevision string) (workspaceflow.State, error) {
	state, err := r.workspaceLifecycle.State(ctx, workspaceID)
	if err != nil {
		return workspaceflow.State{}, err
	}
	if state.Status == "" {
		state.Status = workspaceflow.StatusInactive
	}
	if state.WorkspaceID == "" {
		state.WorkspaceID = string(workspaceID)
	}
	if state.DefinitionRevision == "" {
		if definitionRevision == "" {
			if desired, currentErr := r.workspaces.Current(workspaceID); currentErr == nil {
				definitionRevision = desired.Revision
			}
		}
		state.DefinitionRevision = definitionRevision
	}
	return state, nil
}

func marshalWorkspaceState(state workspaceflow.State) (string, error) {
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

func (r *Runtime) SetWorkspaceNotifier(notify func(string, uint64)) {
	r.workspaceStateMu.Lock()
	defer r.workspaceStateMu.Unlock()
	r.onWorkspaceChanged = notify
}

func (r *Runtime) workspaceRevision() uint64 {
	r.workspaceStateMu.RLock()
	defer r.workspaceStateMu.RUnlock()
	return r.workspaceStateRevision
}

func (r *Runtime) notifyWorkspaceChanged(workspaceID string) {
	r.workspaceStateMu.Lock()
	r.workspaceStateRevision++
	revision := r.workspaceStateRevision
	notify := r.onWorkspaceChanged
	r.workspaceStateMu.Unlock()
	if notify != nil {
		notify(workspaceID, revision)
	}
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
	payload["workspaceStateRevision"] = r.workspaceRevision()
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
			changed, err := r.workspaces.Poll()
			if err != nil {
				if report != nil {
					report(err)
				}
				continue
			}
			if changed {
				r.notifyWorkspaceChanged("")
			}
		}
	}
}
