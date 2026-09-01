package fake

import (
	"context"
	"testing"

	"github.com/Homiakus/HWS/internal/domain"
)

func TestCloseOnlyRemovesManagedResources(t *testing.T) {
	desktop := NewDesktop()
	desired := domain.DesiredState{WorkspaceID: "w", Revision: "1"}
	managed := domain.ResourceSpec{ID: "managed", Kind: domain.ResourceProcess, Ownership: domain.OwnershipManaged, Executable: "m"}
	adopted := domain.ResourceSpec{ID: "adopted", Kind: domain.ResourceProcess, Ownership: domain.OwnershipAdopted, Executable: "a"}

	managedObs, _ := desktop.Ensure(context.Background(), desired, managed)
	adoptedObs, _ := desktop.Ensure(context.Background(), desired, adopted)
	if err := desktop.Close(context.Background(), desired, managed, managedObs); err != nil {
		t.Fatal(err)
	}
	if err := desktop.Close(context.Background(), desired, adopted, adoptedObs); err != nil {
		t.Fatal(err)
	}

	observed, _ := desktop.Observe(context.Background(), desired)
	if _, ok := observed.Resources["managed"]; ok {
		t.Fatal("managed resource should be removed")
	}
	if _, ok := observed.Resources["adopted"]; !ok {
		t.Fatal("adopted resource must not be removed")
	}
}
