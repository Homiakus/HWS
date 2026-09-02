package shelldesktop

import (
	"context"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/Homiakus/HWS/internal/surface"
)

type placementActions struct {
	reader   *fakeReader
	requests []shellaction.Request
}

func (a *placementActions) Request(_ context.Context, request shellaction.Request) (shellaction.Result, error) {
	a.requests = append(a.requests, request)
	if request.Kind == shellaction.KindPlaceWindow {
		a.reader.mu.Lock()
		for ai := range a.reader.shell.Apps {
			for wi := range a.reader.shell.Apps[ai].Windows {
				window := &a.reader.shell.Apps[ai].Windows[wi]
				if window.ID == request.WindowID {
					window.MonitorRef = request.MonitorRef
					window.WorkspaceID = "workspace:2"
					window.Frame = request.Rect
				}
			}
		}
		a.reader.mu.Unlock()
	}
	return shellaction.Result{Schema: 1, ID: request.ID, Success: true, Changed: request.Kind == shellaction.KindPlaceWindow}, nil
}

func TestEnsurePlacementConvergesAgainstFreshShellSnapshot(t *testing.T) {
	reader := &fakeReader{
		snapshot: surface.Snapshot{Surfaces: []surface.ApplicationSurface{{
			AppID:        "dev.zed.Zed.desktop",
			DesktopAppID: "dev.zed.Zed.desktop",
			Windows:      []surface.Window{{ID: "window:7"}},
		}}},
		shell: gnomeshell.Snapshot{
			Schema:     1,
			Revision:   1,
			CapturedAt: time.Now(),
			Topology:   testTopology(),
			Apps: []gnomeshell.Application{{
				AppID:        "dev.zed.Zed.desktop",
				DesktopAppID: "dev.zed.Zed.desktop",
				Windows: []gnomeshell.Window{{
					ID:          "window:7",
					MonitorRef:  "monitor:0",
					WorkspaceID: "workspace:0",
					Frame:       domain.LogicalRect{X: 10, Y: 40, Width: 1000, Height: 700},
				}},
			}},
		},
	}
	actions := &placementActions{reader: reader}
	desktop := New(reader, actions)
	desktop.poll = time.Millisecond
	desired := managedDesired()
	desired.Resources[0].Placement = &domain.PlacementIntent{
		MonitorRole: "secondary",
		Workspace:   2,
		Rect:        domain.NormalizedRect{X: 0.25, Y: 0.1, Width: 0.5, Height: 0.8},
	}

	observation, err := desktop.Ensure(context.Background(), desired, desired.Resources[0])
	if err != nil {
		t.Fatal(err)
	}
	if !observation.Ready || observation.Placement == nil || !observation.Placement.Reached {
		t.Fatalf("placement did not converge: %#v", observation)
	}
	if len(actions.requests) != 2 || actions.requests[0].Kind != shellaction.KindEnsureDesktopApp || actions.requests[1].Kind != shellaction.KindPlaceWindow {
		t.Fatalf("unexpected action sequence: %#v", actions.requests)
	}
	place := actions.requests[1]
	if place.TopologyRevision != "topology:7" || place.MonitorRef != "monitor:1" || place.MonitorIndex != 1 || place.TargetWorkspace != 2 {
		t.Fatalf("unexpected placement request: %#v", place)
	}
}
