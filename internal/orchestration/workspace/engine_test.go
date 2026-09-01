package workspace

import (
	"context"
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

func TestCloseDoesNotDestroyAdoptedResource(t *testing.T) {
	lifecycle, desktop, desired := lifecycleFixture(t)
	desktop.Seed(desired.WorkspaceID, domain.ResourceObservation{
		ResourceID: "shared", Present: true, Ready: true, Ownership: domain.OwnershipAdopted,
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
	if _, ok := observed.Resources["shared"]; !ok {
		t.Fatal("adopted resource was removed")
	}
}
