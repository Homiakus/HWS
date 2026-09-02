package shelldesktop

import (
	"context"
	"sync"
	"testing"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/Homiakus/HWS/internal/surface"
)

type fakeReader struct {
	mu       sync.Mutex
	snapshot surface.Snapshot
}

func (r *fakeReader) SurfaceSnapshot() surface.Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot.Clone()
}

func (r *fakeReader) setApplication(app surface.ApplicationSurface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Surfaces = []surface.ApplicationSurface{app}
}

type fakeActions struct {
	reader   *fakeReader
	requests []shellaction.Request
	changed  bool
}

func (a *fakeActions) Request(_ context.Context, request shellaction.Request) (shellaction.Result, error) {
	a.requests = append(a.requests, request)
	if request.Kind == shellaction.KindEnsureDesktopApp {
		a.reader.setApplication(surface.ApplicationSurface{
			AppID:        surface.ApplicationID(request.DesktopAppID),
			DesktopAppID: request.DesktopAppID,
			Windows:      []surface.Window{{ID: "window:7"}},
		})
	}
	return shellaction.Result{Schema: shellaction.SchemaVersion, ID: request.ID, Success: true, Changed: a.changed}, nil
}

func managedDesired() domain.DesiredState {
	return domain.DesiredState{
		WorkspaceID: "dev",
		Revision:    "v1",
		Resources: []domain.ResourceSpec{{
			ID:           "editor",
			Kind:         domain.ResourceDesktopApp,
			Required:     true,
			Ownership:    domain.OwnershipManaged,
			DesktopAppID: "dev.zed.Zed.desktop",
		}},
	}
}

func TestEnsureMarksOnlyNewlyLaunchedApplicationManaged(t *testing.T) {
	reader := &fakeReader{}
	actions := &fakeActions{reader: reader, changed: true}
	desktop := New(reader, actions)
	desired := managedDesired()
	observation, err := desktop.Ensure(context.Background(), desired, desired.Resources[0])
	if err != nil {
		t.Fatal(err)
	}
	if observation.Ownership != domain.OwnershipManaged || !observation.Ready {
		t.Fatalf("unexpected managed observation: %#v", observation)
	}
}

func TestPreexistingManagedResourceIsObservedAsAdopted(t *testing.T) {
	reader := &fakeReader{}
	reader.setApplication(surface.ApplicationSurface{
		AppID:        "dev.zed.Zed.desktop",
		DesktopAppID: "dev.zed.Zed.desktop",
		Windows:      []surface.Window{{ID: "window:1"}},
	})
	actions := &fakeActions{reader: reader}
	desktop := New(reader, actions)
	desired := managedDesired()
	observed, err := desktop.Observe(context.Background(), desired)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Resources["editor"].Ownership != domain.OwnershipAdopted {
		t.Fatalf("preexisting application was incorrectly claimed as managed: %#v", observed.Resources["editor"])
	}
	if err := desktop.Close(context.Background(), desired, desired.Resources[0], observed.Resources["editor"]); err != nil {
		t.Fatal(err)
	}
	if len(actions.requests) != 0 {
		t.Fatalf("adopted application received close request: %#v", actions.requests)
	}
}

func TestManagedCloseUsesObservedWindowIdentity(t *testing.T) {
	reader := &fakeReader{}
	actions := &fakeActions{reader: reader, changed: true}
	desktop := New(reader, actions)
	desired := managedDesired()
	observation, err := desktop.Ensure(context.Background(), desired, desired.Resources[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := desktop.Close(context.Background(), desired, desired.Resources[0], observation); err != nil {
		t.Fatal(err)
	}
	if len(actions.requests) != 2 || actions.requests[1].Kind != shellaction.KindCloseWindow || actions.requests[1].WindowID != "window:7" {
		t.Fatalf("unexpected requests: %#v", actions.requests)
	}
}
