package contexttree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManagerConfiguresDefaultHierarchyAndResolvesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hierarchy.json")
	manager := &Manager{}
	if err := manager.Configure(path); err != nil {
		t.Fatal(err)
	}
	snapshot, ok := manager.Snapshot()
	if !ok || snapshot.RootID != "root" || len(snapshot.Nodes) < 5 {
		t.Fatalf("unexpected snapshot: %#v", snapshot)
	}
	resolved, err := manager.Path("system-network")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 3 || resolved[0].ID != "root" || resolved[2].ID != "system-network" {
		t.Fatalf("unexpected path: %#v", resolved)
	}
}

func TestInvalidReloadKeepsLastKnownGoodTree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hierarchy.json")
	manager := &Manager{}
	if err := manager.Configure(path); err != nil {
		t.Fatal(err)
	}
	before, _ := manager.Snapshot()
	if err := os.WriteFile(path, []byte(`{"schema":1,"nodes":[{"id":"root","title":""}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := manager.Reload()
	if err == nil || changed {
		t.Fatalf("invalid hierarchy unexpectedly accepted: changed=%v err=%v", changed, err)
	}
	after, ok := manager.Snapshot()
	if !ok || after.Revision != before.Revision || len(after.Nodes) != len(before.Nodes) {
		t.Fatalf("last-known-good hierarchy was lost: before=%#v after=%#v", before, after)
	}
	if manager.LastError() == nil {
		t.Fatal("reload diagnostic was not retained")
	}
}
