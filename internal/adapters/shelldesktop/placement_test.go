package shelldesktop

import (
	"testing"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
)

func testTopology() gnomeshell.Topology {
	return gnomeshell.Topology{
		Revision:          "topology:7",
		PrimaryMonitorRef: "monitor:0",
		Monitors: []gnomeshell.Monitor{
			{Ref: "monitor:0", Index: 0, Primary: true, Scale: 1, Geometry: domain.LogicalRect{X: 0, Y: 0, Width: 1920, Height: 1080}, WorkArea: domain.LogicalRect{X: 0, Y: 32, Width: 1920, Height: 1048}},
			{Ref: "monitor:1", Index: 1, Scale: 1.5, Geometry: domain.LogicalRect{X: 1920, Y: 0, Width: 1707, Height: 960}, WorkArea: domain.LogicalRect{X: 1920, Y: 0, Width: 1707, Height: 960}},
		},
	}
}

func TestResolvePlacementUsesLogicalWorkArea(t *testing.T) {
	intent := &domain.PlacementIntent{
		MonitorRole: "secondary",
		Workspace:   2,
		Rect:        domain.NormalizedRect{X: 0.25, Y: 0.1, Width: 0.5, Height: 0.8},
	}
	got, err := resolvePlacement(intent, testTopology())
	if err != nil {
		t.Fatal(err)
	}
	if got.TopologyRevision != "topology:7" || got.MonitorRef != "monitor:1" || got.MonitorIndex != 1 || got.Workspace != 2 {
		t.Fatalf("unexpected target: %#v", got)
	}
	if got.Rect != (domain.LogicalRect{X: 2347, Y: 96, Width: 854, Height: 768}) {
		t.Fatalf("rect=%#v", got.Rect)
	}
}

func TestPlacementReachedRejectsStaleTopology(t *testing.T) {
	target := domain.ResolvedPlacement{
		TopologyRevision: "topology:7",
		MonitorRef:       "monitor:0",
		Workspace:        1,
		Rect:             domain.LogicalRect{X: 10, Y: 40, Width: 800, Height: 600},
	}
	window := gnomeshell.Window{
		MonitorRef:  "monitor:0",
		WorkspaceID: "workspace:1",
		Frame:       domain.LogicalRect{X: 11, Y: 39, Width: 801, Height: 599},
	}
	if !placementReached(target, window, "topology:7") {
		t.Fatal("placement within tolerance should be reached")
	}
	if placementReached(target, window, "topology:8") {
		t.Fatal("stale topology must never be accepted")
	}
}

func TestResolvePlacementFailsClosedWithoutRequestedMonitor(t *testing.T) {
	_, err := resolvePlacement(&domain.PlacementIntent{
		MonitorRole: "monitor:9",
		Rect:        domain.NormalizedRect{Width: 1, Height: 1},
	}, testTopology())
	if err == nil {
		t.Fatal("missing monitor should fail closed")
	}
}
