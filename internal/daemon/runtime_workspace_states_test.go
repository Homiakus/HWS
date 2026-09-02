package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	workspaceflow "github.com/Homiakus/HWS/internal/orchestration/workspace"
)

func TestWorkspaceStatesProjectionNormalizesInactiveAndSorts(t *testing.T) {
	runtime := NewRuntime(nil)
	workspacePath := filepath.Join(t.TempDir(), "workspaces.json")
	if err := runtime.ConfigureWorkspaces(workspacePath); err != nil {
		t.Fatal(err)
	}
	const source = `{
  "schema":1,
  "active":{"zeta":"v2","alpha":"v1"},
  "workspaces":[
    {"id":"zeta","revision":"v2","resources":[]},
    {"id":"alpha","revision":"v1","resources":[]}
  ]
}`
	if err := os.WriteFile(workspacePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.workspaces.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.OpenWorkspaceLifecycle(filepath.Join(t.TempDir(), "axiom")); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()

	raw, err := runtime.WorkspaceStatesJSON()
	if err != nil {
		t.Fatal(err)
	}
	var snapshot WorkspaceStatesSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Schema != workspaceStatesSchemaVersion || snapshot.Revision == 0 || snapshot.CatalogRevision == 0 {
		t.Fatalf("invalid snapshot metadata: %#v", snapshot)
	}
	if len(snapshot.States) != 2 {
		t.Fatalf("states=%d want=2", len(snapshot.States))
	}
	if snapshot.States[0].WorkspaceID != "alpha" || snapshot.States[1].WorkspaceID != "zeta" {
		t.Fatalf("states are not sorted: %#v", snapshot.States)
	}
	if snapshot.States[0].Status != workspaceflow.StatusInactive || snapshot.States[0].DefinitionRevision != "v1" {
		t.Fatalf("alpha not normalized: %#v", snapshot.States[0])
	}
	if snapshot.States[1].Status != workspaceflow.StatusInactive || snapshot.States[1].DefinitionRevision != "v2" {
		t.Fatalf("zeta not normalized: %#v", snapshot.States[1])
	}
}

func TestWorkspaceMutationAdvancesProjectionRevisionAndNotifies(t *testing.T) {
	runtime := NewRuntime(nil)
	workspacePath := filepath.Join(t.TempDir(), "workspaces.json")
	if err := runtime.ConfigureWorkspaces(workspacePath); err != nil {
		t.Fatal(err)
	}
	const source = `{
  "schema":1,
  "active":{"dev":"v1"},
  "workspaces":[{"id":"dev","revision":"v1","resources":[]}]
}`
	if err := os.WriteFile(workspacePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.workspaces.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.OpenWorkspaceLifecycle(filepath.Join(t.TempDir(), "axiom")); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()

	before := runtime.workspaceRevision()
	var gotID string
	var gotRevision uint64
	runtime.SetWorkspaceNotifier(func(workspaceID string, revision uint64) {
		gotID = workspaceID
		gotRevision = revision
	})

	if _, err := runtime.SuspendWorkspaceJSON("dev"); err != nil {
		t.Fatal(err)
	}
	if gotID != "dev" || gotRevision <= before || runtime.workspaceRevision() != gotRevision {
		t.Fatalf("notification id=%q revision=%d before=%d current=%d", gotID, gotRevision, before, runtime.workspaceRevision())
	}
}
