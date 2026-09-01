package surfacehub

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Homiakus/HWS/internal/panel"
	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

type ActionSender interface {
	Send(providerID string, command any) error
}

type Hub struct {
	registry *providers.Registry
	sender   ActionSender
	revision atomic.Uint64
	mu       sync.Mutex
	changed  []chan uint64
}

func New(registry *providers.Registry, sender ActionSender) *Hub {
	if registry == nil {
		registry = providers.NewRegistry()
	}
	return &Hub{registry: registry, sender: sender}
}

func (h *Hub) Ingest(snapshot providers.Snapshot) error {
	if err := h.registry.Ingest(snapshot); err != nil {
		return err
	}
	rev := h.revision.Add(1)
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.changed {
		select {
		case ch <- rev:
		default:
		}
	}
	return nil
}

func (h *Hub) Subscribe() <-chan uint64 {
	ch := make(chan uint64, 1)
	h.mu.Lock()
	h.changed = append(h.changed, ch)
	h.mu.Unlock()
	return ch
}

func (h *Hub) Revision() uint64 { return h.revision.Load() }

func (h *Hub) Surfaces(now time.Time) []surface.ApplicationSurface {
	return h.registry.Apply(nil, now)
}

func (h *Hub) PanelSnapshot(maxSegments int, now time.Time) panel.Snapshot {
	return panel.Project(h.Surfaces(now), maxSegments, h.Revision(), now)
}

func (h *Hub) ActivateView(appID surface.ApplicationID, viewID surface.ViewID) error {
	return h.sendViewAction(appID, viewID, "activateView")
}

func (h *Hub) CloseView(appID surface.ApplicationID, viewID surface.ViewID) error {
	return h.sendViewAction(appID, viewID, "closeView")
}

func (h *Hub) sendViewAction(appID surface.ApplicationID, viewID surface.ViewID, action string) error {
	if h.sender == nil {
		return fmt.Errorf("surface action transport unavailable")
	}
	for _, app := range h.Surfaces(time.Now()) {
		if app.AppID != appID {
			continue
		}
		for _, window := range app.Windows {
			for _, view := range window.Views {
				if view.ID != viewID {
					continue
				}
				command := map[string]any{"type": action}
				switch {
				case strings.HasPrefix(view.ProviderID, "browser:"):
					id, err := strconv.ParseInt(strings.TrimPrefix(string(view.ID), "tab:"), 10, 64)
					if err != nil {
						return fmt.Errorf("invalid browser view id %q", view.ID)
					}
					command["tabId"] = id
				case view.ProviderID == "vscode":
					if view.ResourceRef == "" {
						return fmt.Errorf("vscode view %q has no resource", view.ID)
					}
					command["resource"] = view.ResourceRef
				default:
					return fmt.Errorf("provider %q does not support view actions", view.ProviderID)
				}
				return h.sender.Send(view.ProviderID, command)
			}
		}
		return fmt.Errorf("view %q not found", viewID)
	}
	return fmt.Errorf("application %q not found", appID)
}
