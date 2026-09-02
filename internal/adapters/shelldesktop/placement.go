package shelldesktop

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
)

const placementTolerance = 2

func resolvePlacement(intent *domain.PlacementIntent, topology gnomeshell.Topology) (domain.ResolvedPlacement, error) {
	if intent == nil {
		return domain.ResolvedPlacement{}, fmt.Errorf("shell desktop: placement intent is required")
	}
	if err := topology.Validate(); err != nil {
		return domain.ResolvedPlacement{}, fmt.Errorf("shell desktop: topology unavailable: %w", err)
	}
	monitor, err := resolveMonitor(strings.TrimSpace(intent.MonitorRole), topology)
	if err != nil {
		return domain.ResolvedPlacement{}, err
	}
	area := monitor.WorkArea
	if !area.Valid() {
		area = monitor.Geometry
	}
	rect := domain.LogicalRect{
		X:      area.X + int(math.Round(intent.Rect.X*float64(area.Width))),
		Y:      area.Y + int(math.Round(intent.Rect.Y*float64(area.Height))),
		Width:  int(math.Round(intent.Rect.Width * float64(area.Width))),
		Height: int(math.Round(intent.Rect.Height * float64(area.Height))),
	}
	if rect.Width < 1 {
		rect.Width = 1
	}
	if rect.Height < 1 {
		rect.Height = 1
	}
	if rect.X+rect.Width > area.X+area.Width {
		rect.Width = area.X + area.Width - rect.X
	}
	if rect.Y+rect.Height > area.Y+area.Height {
		rect.Height = area.Y + area.Height - rect.Y
	}
	if !rect.Valid() {
		return domain.ResolvedPlacement{}, fmt.Errorf("shell desktop: normalized placement resolved outside monitor work area")
	}
	return domain.ResolvedPlacement{
		TopologyRevision: topology.Revision,
		MonitorRef:       monitor.Ref,
		MonitorIndex:     monitor.Index,
		Workspace:        intent.Workspace,
		Rect:             rect,
	}, nil
}

func resolveMonitor(role string, topology gnomeshell.Topology) (gnomeshell.Monitor, error) {
	monitors := append([]gnomeshell.Monitor(nil), topology.Monitors...)
	sort.Slice(monitors, func(i, j int) bool { return monitors[i].Index < monitors[j].Index })
	if role == "" || role == "primary" {
		for _, monitor := range monitors {
			if monitor.Ref == topology.PrimaryMonitorRef || monitor.Primary {
				return monitor, nil
			}
		}
	}
	if role == "secondary" {
		for _, monitor := range monitors {
			if monitor.Ref != topology.PrimaryMonitorRef && !monitor.Primary {
				return monitor, nil
			}
		}
		return gnomeshell.Monitor{}, fmt.Errorf("shell desktop: secondary monitor is unavailable")
	}
	if strings.HasPrefix(role, "monitor:") {
		index, err := strconv.Atoi(strings.TrimPrefix(role, "monitor:"))
		if err != nil || index < 0 {
			return gnomeshell.Monitor{}, fmt.Errorf("shell desktop: invalid monitor role %q", role)
		}
		for _, monitor := range monitors {
			if monitor.Index == index || monitor.Ref == role {
				return monitor, nil
			}
		}
	}
	for _, monitor := range monitors {
		if monitor.Ref == role {
			return monitor, nil
		}
	}
	return gnomeshell.Monitor{}, fmt.Errorf("shell desktop: monitor role %q is unavailable", role)
}

func placementReached(target domain.ResolvedPlacement, window gnomeshell.Window, topologyRevision string) bool {
	if target.TopologyRevision == "" || topologyRevision != target.TopologyRevision {
		return false
	}
	if window.MonitorRef != target.MonitorRef || window.WorkspaceID != fmt.Sprintf("workspace:%d", target.Workspace) {
		return false
	}
	return near(window.Frame.X, target.Rect.X) &&
		near(window.Frame.Y, target.Rect.Y) &&
		near(window.Frame.Width, target.Rect.Width) &&
		near(window.Frame.Height, target.Rect.Height)
}

func near(a, b int) bool {
	delta := a - b
	if delta < 0 {
		delta = -delta
	}
	return delta <= placementTolerance
}

func anchorWindow(app gnomeshell.Application) (gnomeshell.Window, bool) {
	if len(app.Windows) == 0 {
		return gnomeshell.Window{}, false
	}
	windows := append([]gnomeshell.Window(nil), app.Windows...)
	sort.Slice(windows, func(i, j int) bool { return windows[i].ID < windows[j].ID })
	return windows[0], true
}

func findShellApplication(snapshot gnomeshell.Snapshot, desktopAppID string) (gnomeshell.Application, bool) {
	for _, app := range snapshot.Apps {
		if app.DesktopAppID == desktopAppID || app.AppID == desktopAppID {
			return app, true
		}
	}
	return gnomeshell.Application{}, false
}
