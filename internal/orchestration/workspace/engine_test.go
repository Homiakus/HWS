package workspace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Homiakus/HWS/internal/adapters/fake"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
	"github.com/Homiakus/HWS/internal/domain"
)

func lifecycleFixture(t *testing.T) (*Lifecycle, *fake.Desktop, domain.DesiredState) {
	t.Helper()
	desired := domain.DesiredState{WorkspaceID: "local-dev", Revision: "v1", Resources: []domain.ResourceSpec{
		{ID: "editor", Kind: domain.ResourceDesktopApp, Required: true, Ownership: domain.OwnershipManaged, DesktopAppID: "dev.zed.Zed.desktop"},
		{ID: "terminal", Kind: domain.ResourceTerminal, Required: true, Ownership: domain.OwnershipManaged, Executable: "bash"},
	}}
	definitions := catalog.NewMemory()
	if err := definitions.Put(desired); err != nil {
		t.Fatal(err)
	}
	desktop := fake.NewDesktop()
	lifecycle, err := Open(definitions, reconcile.New(desktop))
	if err != nil {
		t.Fatal(err)
	}
	return lifecycle, desktop, desired
}

func TestActivateReachesActive(t *testing.T) {
	lifecycle, _, desired := lifecycleFixture(t)
	ctx := context.Background()
	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:local-dev:v1:1"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	state, err := lifecycle.State(ctx, desired.WorkspaceID)
	if err != nil {
		t.Fatalf("State: %v", err)
	}
	if state.Status != StatusActive || state.ReachedRequired != 2 || state.TotalRequired != 2 {
		t.Fatalf("unexpected state: %#v", state)
	}
}

func TestRepeatedActivateDoesNotDuplicateEnsures(t *testing.T) {
	lifecycle, desktop, desired := lifecycleFixture(t)
	ctx := context.Background()
	key := "activate:local-dev:v1:repeat"

	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, key); err != nil {
		t.Fatalf("first Activate: %v", err)
	}
	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, key); err != nil {
		t.Fatalf("second Activate: %v", err)
	}

	if got := desktop.EnsureAttempts("editor"); got != 1 {
		t.Fatalf("editor ensure attempts=%d want=1", got)
	}
	if got := desktop.EnsureAttempts("terminal"); got != 1 {
		t.Fatalf("terminal ensure attempts=%d want=1", got)
	}
}

func TestRecoverMovesDegradedWorkspaceToActive(t *testing.T) {
	lifecycle, desktop, desired := lifecycleFixture(t)
	desktop.FailEnsure("terminal", 1)
	ctx := context.Background()

	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:local-dev:v1:degraded"); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	state, err := lifecycle.State(ctx, desired.WorkspaceID)
	if err != nil {
		t.Fatalf("State after activate: %v", err)
	}
	if state.Status != StatusDegraded {
		t.Fatalf("status=%s want=%s", state.Status, StatusDegraded)
	}

	if err := lifecycle.Recover(ctx, desired.WorkspaceID, "recover:local-dev:v1:1"); err != nil {
		t.Fatalf("Recover: %v", err)
	}
	state, err = lifecycle.State(ctx, desired.WorkspaceID)
	if err != nil {
		t.Fatalf("State after recover: %v", err)
	}
	if state.Status != StatusActive || state.ReachedRequired != 2 {
		t.Fatalf("unexpected recovered state: %#v", state)
	}
}

func TestProductionLifecycleSurvivesStoreReopen(t *testing.T) {
	ctx := context.Background()
	desired := domain.DesiredState{WorkspaceID: "durable-dev", Revision: "v1", Resources: []domain.ResourceSpec{
		{ID: "editor", Kind: domain.ResourceDesktopApp, Required: true, Ownership: domain.OwnershipManaged, DesktopAppID: "dev.zed.Zed.desktop"},
		{ID: "terminal", Kind: domain.ResourceTerminal, Required: true, Ownership: domain.OwnershipManaged, Executable: "bash"},
	}}
	definitions := catalog.NewMemory()
	if err := definitions.Put(desired); err != nil {
		t.Fatal(err)
	}
	desktop := fake.NewDesktop()
	storePath := filepath.Join(t.TempDir(), "axiom")

	first, err := OpenProduction(storePath, definitions, reconcile.New(desktop))
	if err != nil {
		t.Fatalf("OpenProduction first: %v", err)
	}
	if err := first.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:durable-dev:v1:1"); err != nil {
		_ = first.Shutdown()
		t.Fatalf("Activate: %v", err)
	}
	if err := first.Shutdown(); err != nil {
		t.Fatalf("Shutdown first: %v", err)
	}

	second, err := OpenProduction(storePath, definitions, reconcile.New(desktop))
	if err != nil {
		t.Fatalf("OpenProduction second: %v", err)
	}
	defer func() {
		if err := second.Shutdown(); err != nil {
			t.Errorf("Shutdown second: %v", err)
		}
	}()

	persisted, err := second.State(ctx, desired.WorkspaceID)
	if err != nil {
		t.Fatalf("State after reopen: %v", err)
	}
	if persisted.Status != StatusActive || persisted.ReachedRequired != 2 || persisted.TotalRequired != 2 {
		t.Fatalf("unexpected persisted state: %#v", persisted)
	}

	if err := second.Recover(ctx, desired.WorkspaceID, "recover:durable-dev:v1:1"); err != nil {
		t.Fatalf("Recover after reopen: %v", err)
	}
	if got := desktop.EnsureAttempts("editor"); got != 1 {
		t.Fatalf("editor ensure attempts=%d want=1 after recovery", got)
	}
	if got := desktop.EnsureAttempts("terminal"); got != 1 {
		t.Fatalf("terminal ensure attempts=%d want=1 after recovery", got)
	}
}

func TestCloseDoesNotDestroyAdoptedResource(t *testing.T) {
	lifecycle, desktop, desired := lifecycleFixture(t)
	adopted := domain.ResourceSpec{
		ID:         "shared",
		Kind:       domain.ResourceProcess,
		Ownership:  domain.OwnershipAdopted,
		Executable: "shared-tool",
	}
	desired.Resources = append(desired.Resources, adopted)

	definitions := catalog.NewMemory()
	if err := definitions.Put(desired); err != nil {
		t.Fatal(err)
	}
	var err error
	lifecycle, err = Open(definitions, reconcile.New(desktop))
	if err != nil {
		t.Fatal(err)
	}
	desktop.Seed(desired.WorkspaceID, domain.ResourceObservation{
		ResourceID: adopted.ID,
		Present:    true,
		Ready:      true,
		Ownership:  domain.OwnershipAdopted,
		SessionRef: "fake:shared",
	})

	ctx := context.Background()
	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:local-dev:v1:2"); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Close(ctx, desired.WorkspaceID, "close:local-dev:v1:1"); err != nil {
		t.Fatal(err)
	}
	observed, err := desktop.Observe(ctx, desired)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := observed.Resources[adopted.ID]; !ok {
		t.Fatal("adopted resource was removed")
	}
}
