package workspace

import (
	"context"
	"testing"

	"github.com/Homiakus/HWS/internal/adapters/fake"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
)

func TestStateNormalizesNeverStartedExecutionToInactive(t *testing.T) {
	lifecycle, err := Open(catalog.NewMemory(), reconcile.New(fake.NewDesktop()))
	if err != nil {
		t.Fatal(err)
	}
	state, err := lifecycle.State(context.Background(), "never-started")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != StatusInactive || state.WorkspaceID != "never-started" {
		t.Fatalf("unexpected state: %#v", state)
	}
}
