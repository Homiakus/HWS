package domain

import "testing"

func TestBuildReconcilePlanIsDeterministicAndSkipsExternal(t *testing.T) {
	desired := DesiredState{WorkspaceID: "w", Revision: "1", Resources: []ResourceSpec{
		{ID: "z", Kind: ResourceProcess, Ownership: OwnershipManaged, Executable: "z"},
		{ID: "a", Kind: ResourceProcess, Ownership: OwnershipManaged, Executable: "a"},
		{ID: "external", Kind: ResourceProcess, Ownership: OwnershipExternal, Executable: "x"},
	}}
	observed := ObservedState{WorkspaceID: "w", Resources: map[ResourceID]ResourceObservation{}}

	plan := BuildReconcilePlan(desired, observed)
	if len(plan.Actions) != 2 || plan.Actions[0].Resource.ID != "a" || plan.Actions[1].Resource.ID != "z" {
		t.Fatalf("unexpected actions: %#v", plan.Actions)
	}
	if len(plan.UnmanagedUnreached) != 1 || plan.UnmanagedUnreached[0] != "external" {
		t.Fatalf("unexpected unmanaged: %#v", plan.UnmanagedUnreached)
	}
}

func TestEvaluateReconcileCountsRequiredResources(t *testing.T) {
	desired := DesiredState{WorkspaceID: "w", Revision: "1", Resources: []ResourceSpec{
		{ID: "editor", Kind: ResourceDesktopApp, Required: true, Ownership: OwnershipManaged, DesktopAppID: "dev.zed.Zed.desktop"},
		{ID: "terminal", Kind: ResourceTerminal, Required: true, Ownership: OwnershipManaged, Executable: "bash"},
		{ID: "docs", Kind: ResourceDesktopApp, Ownership: OwnershipManaged, DesktopAppID: "firefox.desktop"},
	}}
	observed := ObservedState{WorkspaceID: "w", Resources: map[ResourceID]ResourceObservation{
		"editor": {ResourceID: "editor", Present: true, Ready: true},
		"docs":   {ResourceID: "docs", Present: true, Ready: true},
	}}

	got := EvaluateReconcile(desired, observed)
	if got.RequiredTotal != 2 || got.RequiredReached != 1 || got.OptionalReached != 1 {
		t.Fatalf("unexpected evaluation: %#v", got)
	}
	if len(got.MissingRequired) != 1 || got.MissingRequired[0] != "terminal" {
		t.Fatalf("unexpected missing required: %#v", got.MissingRequired)
	}
}
