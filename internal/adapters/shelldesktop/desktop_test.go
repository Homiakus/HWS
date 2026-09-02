package shelldesktop

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/Homiakus/HWS/internal/surface"
)

type fakeReader struct {
	mu       sync.Mutex
	snapshot surface.Snapshot
	shell    gnomeshell.Snapshot
}

func (r *fakeReader) SurfaceSnapshot() surface.Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshot.Clone()
}

func (r *fakeReader) ShellSnapshot() gnomeshell.Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.shell
	out.Topology.Monitors = append([]gnomeshell.Monitor(nil), r.shell.Topology.Monitors...)
	out.Apps = append([]gnomeshell.Application(nil), r.shell.Apps...)
	for i := range out.Apps {
		out.Apps[i].Windows = append([]gnomeshell.Window(nil), r.shell.Apps[i].Windows...)
	}
	return out
}

func (r *fakeReader) setApplication(app surface.ApplicationSurface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.snapshot.Surfaces = []surface.ApplicationSurface{app}
}

func (r *fakeReader) removeWindow(windowID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for ai := range r.snapshot.Surfaces {
		windows := r.snapshot.Surfaces[ai].Windows[:0]
		for _, window := range r.snapshot.Surfaces[ai].Windows {
			if string(window.ID) != windowID {
				windows = append(windows, window)
			}
		}
		r.snapshot.Surfaces[ai].Windows = windows
	}
}

type fakeActions struct {
	reader    *fakeReader
	requests  []shellaction.Request
	changed   bool
	holdClose bool
}

func (a *fakeActions) Request(_ context.Context, request shellaction.Request) (shellaction.Result, error) {
	a.requests = append(a.requests, request)
	switch request.Kind {
	case shellaction.KindEnsureDesktopApp:
		a.reader.setApplication(surface.ApplicationSurface{
			AppID:        surface.ApplicationID(request.DesktopAppID),
			DesktopAppID: request.DesktopAppID,
			Windows:      []surface.Window{{ID: "window:7"}},
		})
	case shellaction.KindCloseWindow:
		if !a.holdClose {
			a.reader.removeWindow(request.WindowID)
		}
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

func TestManagedCloseUsesObservedWindowIdentityAndWaitsForAbsence(t *testing.T) {
	reader := &fakeReader{}
	actions := &fakeActions{reader: reader, changed: true}
	desktop := New(reader, actions)
	desktop.poll = time.Millisecond
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

func TestManagedCloseFailsIfClientRefusesToDisappear(t *testing.T) {
	reader := &fakeReader{}
	actions := &fakeActions{reader: reader, changed: true}
	desktop := New(reader, actions)
	desktop.poll = time.Millisecond
	desired := managedDesired()
	observation, err := desktop.Ensure(context.Background(), desired, desired.Resources[0])
	if err != nil {
		t.Fatal(err)
	}
	actions.holdClose = true
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := desktop.Close(ctx, desired, desired.Resources[0], observation); err == nil {
		t.Fatal("close converged even though the tracked window remained present")
	}
	if !desktop.isManaged(desired.WorkspaceID, desired.Resources[0].ID) {
		t.Fatal("managed evidence was dropped after a non-converged close")
	}
}
