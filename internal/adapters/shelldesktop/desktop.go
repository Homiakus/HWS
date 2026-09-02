package shelldesktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/Homiakus/HWS/internal/surface"
)

type SurfaceReader interface {
	SurfaceSnapshot() surface.Snapshot
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
	observed := domain.ObservedState{
		WorkspaceID: desired.WorkspaceID,
		Resources:   make(map[domain.ResourceID]domain.ResourceObservation),
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
		observed.Resources[resource.ID] = domain.ResourceObservation{
			ResourceID: resource.ID,
			Present:    true,
			Ready:      len(app.Windows) > 0,
			Ownership:  ownership,
			SessionRef: string(sessionRef),
			AppID:      string(app.AppID),
		}
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
		code := result.Code
		if code == "" {
			code = "shell_action_failed"
		}
		return domain.ResourceObservation{}, fmt.Errorf("shell desktop: ensure %s failed (%s): %s", resource.ID, code, result.Message)
	}
	if result.Changed && resource.Ownership == domain.OwnershipManaged {
		d.markManaged(desired.WorkspaceID, resource.ID, true)
	}

	ticker := time.NewTicker(d.poll)
	defer ticker.Stop()
	for {
		observed, err := d.Observe(ctx, desired)
		if err != nil {
			return domain.ResourceObservation{}, err
		}
		if observation, ok := observed.Resources[resource.ID]; ok && observation.Present && observation.Ready {
			return observation, nil
		}
		select {
		case <-ctx.Done():
			return domain.ResourceObservation{}, fmt.Errorf("shell desktop: wait for %s: %w", resource.ID, ctx.Err())
		case <-ticker.C:
		}
	}
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
	d.markManaged(desired.WorkspaceID, resource.ID, false)
	return nil
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
