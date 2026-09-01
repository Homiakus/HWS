package domain

import "testing"

func TestTreePathAndStableChildren(t *testing.T) {
	tree, err := NewTree([]Node{
		{ID: "root", Title: "Home", Kind: NodeCategory},
		{ID: "projects", ParentID: "root", Title: "Projects", Order: 2},
		{ID: "dev", ParentID: "root", Title: "Development", Order: 1},
		{ID: "hws", ParentID: "dev", Title: "HWS", Kind: NodeProject},
		{ID: "develop", ParentID: "hws", Title: "Develop", Kind: NodeTask, WorkspaceID: "local-dev"},
	})
	if err != nil {
		t.Fatalf("NewTree: %v", err)
	}

	children := tree.Children("root")
	if len(children) != 2 || children[0].ID != "dev" || children[1].ID != "projects" {
		t.Fatalf("unexpected child order: %#v", children)
	}

	path, err := tree.Path("develop")
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := []NodeID{"root", "dev", "hws", "develop"}
	if len(path) != len(want) {
		t.Fatalf("path length=%d want=%d", len(path), len(want))
	}
	for i := range want {
		if path[i].ID != want[i] {
			t.Fatalf("path[%d]=%q want=%q", i, path[i].ID, want[i])
		}
	}
}

func TestTreeRejectsCycle(t *testing.T) {
	_, err := NewTree([]Node{
		{ID: "root", Title: "Home"},
		{ID: "a", ParentID: "b", Title: "A"},
		{ID: "b", ParentID: "a", Title: "B"},
	})
	if err == nil {
		t.Fatal("expected cycle error")
	}
}
