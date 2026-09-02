package shelldesktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/providers/gnomeshell"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/Homiakus/HWS/internal/surface"
)

type SurfaceReader interface {
	SurfaceSnapshot() surface.Snapshot
	ShellSnapshot() gnomeshell.Snapshot
}

type ActionRequester interface {
	Request(context.Context, shellaction.Request) (shellaction.Result, error)
}

type Desktop struct {
	reader  SurfaceReader
	actions ActionRequester

	mu      sync.Mutex
	managed map[string]bool
	poll    time.Duration
}

func New(reader SurfaceReader, actions ActionRequester) *Desktop {
	return &Desktop{
		reader:  reader,
		actions: actions,
		managed: make(map[string]bool),
		poll:    50 * time.Millisecond,
	}
}

func resourceKey(workspaceID domain.WorkspaceID, resourceID domain.ResourceID) string {
	return string(workspaceID) + "\x00" + string(resourceID)
}

func (d *Desktop) Observe(_ context.Context, desired domain.DesiredState) (domain.ObservedState, error) {
	if d.reader == nil {
		return domain.ObservedState{}, errors.New("shell desktop: surface reader is unavailable")
	}
	snapshot := d.reader.SurfaceSnapshot()
	shell := d.reader.ShellSnapshot()
	observed := domain.ObservedState{
		WorkspaceID:      desired.WorkspaceID,
		TopologyRevision: shell.Topology.Revision,
		Resources:        make(map[domain.ResourceID]domain.ResourceObservation),
	}
	for _, resource := range desired.Resources {
		if resource.Kind != domain.ResourceDesktopApp {
			continue
		}
		app, ok := findApplication(snapshot, resource.DesktopAppID)
		if !ok {
			continue
		}
		ownership := resource.Ownership
		if resource.Ownership == domain.OwnershipManaged && !d.isManaged(desired.WorkspaceID, resource.ID) {
			// An already-running application is adopted unless this adapter has
			// evidence that HWS launched it during the current daemon session.
			ownership = domain.OwnershipAdopted
		}
		windowIDs := make([]string, 0, len(app.Windows))
		for _, window := range app.Windows {
			windowIDs = append(windowIDs, string(window.ID))
		}
		sessionRef, _ := json.Marshal(windowIDs)
		observation := domain.ResourceObservation{
			ResourceID: resource.ID,
			Present:    true,
			Ready:      len(app.Windows) > 0,
			Ownership:  ownership,
			SessionRef: string(sessionRef),
			AppID:      string(app.AppID),
		}
		if resource.Placement != nil {
			observation.Ready = false
			target, err := resolvePlacement(resource.Placement, shell.Topology)
			if err != nil {
				observation.ReasonCode = "topology_unavailable"
				observed.Resources[resource.ID] = observation
				continue
			}
			shellApp, shellOK := findShellApplication(shell, resource.DesktopAppID)
			window, windowOK := anchorWindow(shellApp)
			placement := &domain.PlacementObservation{
				TopologyRevision: shell.Topology.Revision,
				MonitorRef:       target.MonitorRef,
				Workspace:        target.Workspace,
			}
			if shellOK && windowOK {
				placement.MonitorRef = window.MonitorRef
				placement.Workspace = parseWorkspaceIndex(window.WorkspaceID)
				placement.Rect = window.Frame
				placement.Reached = placementReached(target, window, shell.Topology.Revision)
				observation.Ready = placement.Reached
				if !placement.Reached {
					observation.ReasonCode = "placement_unreached"
				}
			} else {
				observation.ReasonCode = "window_geometry_unavailable"
			}
			observation.Placement = placement
		}
		observed.Resources[resource.ID] = observation
	}
	return observed, nil
}

func (d *Desktop) Ensure(ctx context.Context, desired domain.DesiredState, resource domain.ResourceSpec) (domain.ResourceObservation, error) {
	if resource.Kind != domain.ResourceDesktopApp {
		return domain.ResourceObservation{}, fmt.Errorf("shell desktop: resource %s kind %s is unsupported", resource.ID, resource.Kind)
	}
	if d.actions == nil {
		return domain.ResourceObservation{}, errors.New("shell desktop: action broker is unavailable")
	}
	result, err := d.actions.Request(ctx, shellaction.Request{
		Kind:         shellaction.KindEnsureDesktopApp,
		WorkspaceID:  string(desired.WorkspaceID),
		ResourceID:   string(resource.ID),
		DesktopAppID: resource.DesktopAppID,
	})
	if err != nil {
		return domain.ResourceObservation{}, err
	}
	if !result.Success {
		return domain.ResourceObservation{}, shellActionError("ensure", resource.ID, result)
	}
	if result.Changed && resource.Ownership == domain.OwnershipManaged {
		d.markManaged(desired.WorkspaceID, resource.ID, true)
	}

	if err := d.waitPresent(ctx, resource.DesktopAppID); err != nil {
		return domain.ResourceObservation{}, fmt.Errorf("shell desktop: wait for %s: %w", resource.ID, err)
	}
	if resource.Placement != nil {
		if err := d.ensurePlacement(ctx, desired, resource); err != nil {
			return domain.ResourceObservation{}, err
		}
	}
	return d.waitReady(ctx, desired, resource.ID, "")
}

func (d *Desktop) ensurePlacement(ctx context.Context, desired domain.DesiredState, resource domain.ResourceSpec) error {
	shell := d.reader.ShellSnapshot()
	target, err := resolvePlacement(resource.Placement, shell.Topology)
	if err != nil {
		return err
	}
	app, ok := findShellApplication(shell, resource.DesktopAppID)
	if !ok {
		return fmt.Errorf("shell desktop: native application %s is not observed", resource.DesktopAppID)
	}
	window, ok := anchorWindow(app)
	if !ok {
		return fmt.Errorf("shell desktop: application %s has no placeable window", resource.DesktopAppID)
	}
	if placementReached(target, window, shell.Topology.Revision) {
		return nil
	}
	result, err := d.actions.Request(ctx, shellaction.Request{
		Kind:             shellaction.KindPlaceWindow,
		WorkspaceID:      string(desired.WorkspaceID),
		ResourceID:       string(resource.ID),
		WindowID:         window.ID,
		TopologyRevision: target.TopologyRevision,
		MonitorRef:       target.MonitorRef,
		MonitorIndex:     target.MonitorIndex,
		TargetWorkspace:  target.Workspace,
		Rect:             target.Rect,
	})
	if err != nil {
		return err
	}
	if !result.Success {
		return shellActionError("place", resource.ID, result)
	}
	_, err = d.waitReady(ctx, desired, resource.ID, target.TopologyRevision)
	return err
}

func (d *Desktop) waitPresent(ctx context.Context, desktopAppID string) error {
	ticker := time.NewTicker(d.poll)
	defer ticker.Stop()
	for {
		if app, ok := findApplication(d.reader.SurfaceSnapshot(), desktopAppID); ok && len(app.Windows) > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (d *Desktop) waitReady(ctx context.Context, desired domain.DesiredState, resourceID domain.ResourceID, expectedTopology string) (domain.ResourceObservation, error) {
	ticker := time.NewTicker(d.poll)
	defer ticker.Stop()
	for {
		observed, err := d.Observe(ctx, desired)
		if err != nil {
			return domain.ResourceObservation{}, err
		}
		if expectedTopology != "" && observed.TopologyRevision != expectedTopology {
			return domain.ResourceObservation{}, fmt.Errorf("shell desktop: topology changed from %s to %s", expectedTopology, observed.TopologyRevision)
		}
		if observation, ok := observed.Resources[resourceID]; ok && observation.Present && observation.Ready {
			return observation, nil
		}
		select {
		case <-ctx.Done():
			return domain.ResourceObservation{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func shellActionError(action string, resourceID domain.ResourceID, result shellaction.Result) error {
	code := result.Code
	if code == "" {
		code = "shell_action_failed"
	}
	return fmt.Errorf("shell desktop: %s %s failed (%s): %s", action, resourceID, code, result.Message)
}

func (d *Desktop) Close(ctx context.Context, desired domain.DesiredState, resource domain.ResourceSpec, observed domain.ResourceObservation) error {
	if resource.Kind != domain.ResourceDesktopApp {
		return fmt.Errorf("shell desktop: close resource %s kind %s is unsupported", resource.ID, resource.Kind)
	}
	if resource.Ownership != domain.OwnershipManaged || observed.Ownership != domain.OwnershipManaged {
		return nil
	}
	if d.actions == nil {
		return errors.New("shell desktop: action broker is unavailable")
	}
	var windowIDs []string
	if observed.SessionRef != "" {
		if err := json.Unmarshal([]byte(observed.SessionRef), &windowIDs); err != nil {
			return fmt.Errorf("shell desktop: decode managed window identities: %w", err)
		}
	}
	for _, windowID := range windowIDs {
		result, err := d.actions.Request(ctx, shellaction.Request{
			Kind:        shellaction.KindCloseWindow,
			WorkspaceID: string(desired.WorkspaceID),
			ResourceID:  string(resource.ID),
			WindowID:    windowID,
		})
		if err != nil {
			return err
		}
		if !result.Success {
			return fmt.Errorf("shell desktop: close %s window %s failed (%s): %s", resource.ID, windowID, result.Code, result.Message)
		}
	}
	if err := d.waitWindowsAbsent(ctx, resource.DesktopAppID, windowIDs); err != nil {
		return fmt.Errorf("shell desktop: close %s did not converge: %w", resource.ID, err)
	}
	d.markManaged(desired.WorkspaceID, resource.ID, false)
	return nil
}

func (d *Desktop) waitWindowsAbsent(ctx context.Context, desktopAppID string, windowIDs []string) error {
	if len(windowIDs) == 0 {
		return nil
	}
	target := make(map[string]struct{}, len(windowIDs))
	for _, id := range windowIDs {
		target[id] = struct{}{}
	}
	ticker := time.NewTicker(d.poll)
	defer ticker.Stop()
	for {
		remaining := 0
		if app, ok := findApplication(d.reader.SurfaceSnapshot(), desktopAppID); ok {
			for _, window := range app.Windows {
				if _, tracked := target[string(window.ID)]; tracked {
					remaining++
				}
			}
		}
		if remaining == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (d *Desktop) markManaged(workspaceID domain.WorkspaceID, resourceID domain.ResourceID, value bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := resourceKey(workspaceID, resourceID)
	if value {
		d.managed[key] = true
	} else {
		delete(d.managed, key)
	}
}

func (d *Desktop) isManaged(workspaceID domain.WorkspaceID, resourceID domain.ResourceID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.managed[resourceKey(workspaceID, resourceID)]
}

func findApplication(snapshot surface.Snapshot, desktopAppID string) (surface.ApplicationSurface, bool) {
	for _, app := range snapshot.Surfaces {
		if app.DesktopAppID == desktopAppID || string(app.AppID) == desktopAppID {
			return app, true
		}
	}
	return surface.ApplicationSurface{}, false
}

func parseWorkspaceIndex(value string) int {
	index, err := strconv.Atoi(strings.TrimPrefix(value, "workspace:"))
	if err != nil || index < 0 {
		return -1
	}
	return index
}
