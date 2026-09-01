package reconcile

import (
	"context"
	"testing"

	"github.com/Homiakus/HWS/internal/adapters/fake"
	"github.com/Homiakus/HWS/internal/domain"
)

func testWorkspace() domain.DesiredState {
	return domain.DesiredState{WorkspaceID: "local-dev", Revision: "v1", Resources: []domain.ResourceSpec{
		{ID: "editor", Kind: domain.ResourceDesktopApp, Required: true, Ownership: domain.OwnershipManaged, DesktopAppID: "dev.zed.Zed.desktop"},
		{ID: "terminal", Kind: domain.ResourceTerminal, Required: true, Ownership: domain.OwnershipManaged, Executable: "bash"},
	}}
}

func TestReconcileIsIdempotentAtAdapterBoundary(t *testing.T) {
	desktop := fake.NewDesktop()
	r := New(desktop)
	ctx := context.Background()
	desired := testWorkspace()

	first, err := r.Reconcile(ctx, desired)
	if err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	if first.TargetStatus() != "active" {
		t.Fatalf("status=%s", first.TargetStatus())
	}
	second, err := r.Reconcile(ctx, desired)
	if err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if second.TargetStatus() != "active" {
		t.Fatalf("status=%s", second.TargetStatus())
	}
	if desktop.EnsureAttempts("editor") != 1 || desktop.EnsureAttempts("terminal") != 1 {
		t.Fatalf("ensure attempts editor=%d terminal=%d", desktop.EnsureAttempts("editor"), desktop.EnsureAttempts("terminal"))
	}
}

func TestReconcileReportsPartialFailure(t *testing.T) {
	desktop := fake.NewDesktop()
	desktop.FailEnsure("terminal", 1)
	r := New(desktop)

	result, err := r.Reconcile(context.Background(), testWorkspace())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.TargetStatus() != "degraded" {
		t.Fatalf("status=%s want degraded", result.TargetStatus())
	}
	if result.Evaluation.RequiredReached != 1 || len(result.Failures) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}
