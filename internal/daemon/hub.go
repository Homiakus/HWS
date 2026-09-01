package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/panel"
	"github.com/Homiakus/HWS/internal/panel/dsl"
	"github.com/Homiakus/HWS/internal/providers"
	providerserver "github.com/Homiakus/HWS/internal/providers/server"
	"github.com/Homiakus/HWS/internal/surface"
)

const DefaultPanelSource = `panel "main" {
  edge = "top"
  height = 40
  gap = 6
  overflow = "popover"

  group "applications" {
    source = "running"
    app {
      density = "adaptive"
      min_width = 64
      preferred_width = 156
      max_width = 240
      surfaces {
        mode = "segments"
        max_visible = 4
        overflow = "count"
      }
    }
    on "click" { action = "focus_or_cycle" }
    on "scroll" { action = "cycle_surface" }
    on "middle_click" { action = "new_window" }
    on "secondary_click" { action = "surface_menu" }
  }

  group "system" {
    widget "network" { variant = "mini" }
    widget "audio" { variant = "mini" }
    widget "clock" { format = "HH:mm" }
  }
}
`

type Hub struct {
	mu sync.RWMutex

	registry *providers.Registry
	manager  *dsl.Manager
	actions  *providerserver.Actions
	now      func() time.Time

	configPath string
	configMod  time.Time
	configSize int64

	surfaceSnapshot  surface.Snapshot
	panelSnapshot    panel.Snapshot
	panelRevision    uint64
	configRevision   uint64
	lastSpecRevision uint64

	onPanelChanged       func(uint64)
	onPanelConfigChanged func(uint64)
}

func NewHub(registry *providers.Registry) *Hub {
	if registry == nil {
		registry = providers.NewRegistry()
	}
	return &Hub{registry: registry, manager: &dsl.Manager{}, now: time.Now}
}

func (h *Hub) SetActions(actions *providerserver.Actions) {
	h.mu.Lock()
	h.actions = actions
	h.mu.Unlock()
}

func (h *Hub) SetNotifiers(panelChanged, panelConfigChanged func(uint64)) {
	h.mu.Lock()
	h.onPanelChanged = panelChanged
	h.onPanelConfigChanged = panelConfigChanged
	h.mu.Unlock()
}

func (h *Hub) Ingest(snapshot providers.Snapshot) error {
	if err := h.registry.Ingest(snapshot); err != nil {
		return err
	}
	return h.Refresh()
}

func (h *Hub) Refresh() error {
	now := h.now()
	rich := h.registry.Apply(nil, now)

	h.mu.Lock()
	previous := h.surfaceSnapshot.Clone()
	h.mu.Unlock()

	next, err := surface.NewSnapshot(previous, rich, nil)
	if err != nil {
		return err
	}

	spec, specRevision, valid := h.manager.Current()
	maxSegments := 4
	if valid {
		maxSegments = maxVisibleSegments(spec)
	}

	h.mu.Lock()
	surfaceChanged := next.Revision != h.surfaceSnapshot.Revision
	specChanged := specRevision != h.lastSpecRevision
	h.surfaceSnapshot = next
	h.lastSpecRevision = specRevision
	if surfaceChanged || specChanged || h.panelRevision == 0 {
		h.panelRevision++
	}
	h.panelSnapshot = panel.Project(next.Surfaces, maxSegments, h.panelRevision, now)
	panelRevision := h.panelRevision
	notify := h.onPanelChanged
	h.mu.Unlock()

	if (surfaceChanged || specChanged) && notify != nil {
		notify(panelRevision)
	}
	return nil
}

func maxVisibleSegments(spec panel.Spec) int {
	for _, group := range spec.Groups {
		if group.App != nil {
			return group.App.Surfaces.MaxVisible
		}
	}
	return 4
}

func (h *Hub) Configure(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("panel config path is required")
	}
	if _, _, err := h.manager.Apply([]byte(DefaultPanelSource)); err != nil {
		return fmt.Errorf("compile built-in panel: %w", err)
	}

	h.mu.Lock()
	h.configPath = path
	h.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(DefaultPanelSource), 0o600); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if err := h.ReloadPanel(); err != nil {
		// The built-in last-known-good spec remains active.
		_ = h.Refresh()
		return err
	}
	return nil
}

func (h *Hub) ReloadPanel() error {
	h.mu.RLock()
	path := h.configPath
	h.mu.RUnlock()
	if path == "" {
		return errors.New("panel config path is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, revision, err := h.manager.Apply(data)
	if err != nil {
		return err
	}
	info, statErr := os.Stat(path)

	h.mu.Lock()
	h.configRevision = revision
	if statErr == nil {
		h.configMod = info.ModTime()
		h.configSize = info.Size()
	}
	notify := h.onPanelConfigChanged
	h.mu.Unlock()

	if err := h.Refresh(); err != nil {
		return err
	}
	if notify != nil {
		notify(revision)
	}
	return nil
}

func (h *Hub) PollConfig() error {
	h.mu.RLock()
	path, mod, size := h.configPath, h.configMod, h.configSize
	h.mu.RUnlock()
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() == size && info.ModTime().Equal(mod) {
		return nil
	}
	return h.ReloadPanel()
}

func (h *Hub) RunMaintenance(ctx context.Context, interval time.Duration, report func(error)) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := h.PollConfig(); err != nil && report != nil {
				report(err)
			}
			if err := h.Refresh(); err != nil && report != nil {
				report(err)
			}
		}
	}
}

func (h *Hub) PanelJSON() (string, error) {
	h.mu.RLock()
	snapshot := h.panelSnapshot
	h.mu.RUnlock()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (h *Hub) SpecJSON() (string, error) {
	spec, revision, valid := h.manager.Current()
	payload := struct {
		Revision uint64     `json:"revision"`
		Valid    bool       `json:"valid"`
		Spec     panel.Spec `json:"spec"`
	}{Revision: revision, Valid: valid, Spec: spec}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (h *Hub) ApplicationJSON(appID string) (string, error) {
	h.mu.RLock()
	snapshot := h.surfaceSnapshot.Clone()
	h.mu.RUnlock()
	for _, app := range snapshot.Surfaces {
		if string(app.AppID) == appID {
			data, err := json.Marshal(app)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("application surface %q not found", appID)
}

func (h *Hub) ActivateView(appID, viewID string) error {
	app, view, err := h.findView(appID, viewID)
	if err != nil {
		return err
	}
	if !app.Capabilities[surface.CapabilityViewActivate] {
		return fmt.Errorf("view activation is not supported for %s", appID)
	}
	return h.sendViewCommand(view, "activateView")
}

func (h *Hub) CloseView(appID, viewID string) error {
	app, view, err := h.findView(appID, viewID)
	if err != nil {
		return err
	}
	if !app.Capabilities[surface.CapabilityViewClose] {
		return fmt.Errorf("view close is not supported for %s", appID)
	}
	return h.sendViewCommand(view, "closeView")
}

func (h *Hub) findView(appID, viewID string) (surface.ApplicationSurface, surface.View, error) {
	h.mu.RLock()
	snapshot := h.surfaceSnapshot.Clone()
	h.mu.RUnlock()
	for _, app := range snapshot.Surfaces {
		if string(app.AppID) != appID {
			continue
		}
		for _, window := range app.Windows {
			for _, view := range window.Views {
				if string(view.ID) == viewID {
					return app, view, nil
				}
			}
		}
		return surface.ApplicationSurface{}, surface.View{}, fmt.Errorf("view %q not found for %s", viewID, appID)
	}
	return surface.ApplicationSurface{}, surface.View{}, fmt.Errorf("application surface %q not found", appID)
}

func (h *Hub) sendViewCommand(view surface.View, commandType string) error {
	h.mu.RLock()
	actions := h.actions
	h.mu.RUnlock()
	if actions == nil {
		return errors.New("provider action transport is unavailable")
	}
	if view.ProviderID == "" {
		return errors.New("view has no provider identity")
	}
	command := map[string]any{
		"type":   commandType,
		"viewId": string(view.ID),
	}
	if view.ResourceRef != "" {
		command["resource"] = view.ResourceRef
	}
	if strings.HasPrefix(view.ProviderID, "browser:") {
		id, err := strconv.ParseInt(strings.TrimPrefix(string(view.ID), "tab:"), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid browser tab identity %q: %w", view.ID, err)
		}
		command["tabId"] = id
	}
	return actions.Send(view.ProviderID, command)
}
