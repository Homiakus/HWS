package gnomeshell

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

const (
	SchemaVersion uint32 = 1
	ProviderID           = "gnome-shell"
)

type Snapshot struct {
	Schema     uint32        `json:"schema"`
	Revision   uint64        `json:"revision"`
	CapturedAt time.Time     `json:"capturedAt"`
	Apps       []Application `json:"apps"`
}

type Application struct {
	AppID        string   `json:"appId"`
	Name         string   `json:"name"`
	IconName     string   `json:"iconName,omitempty"`
	DesktopAppID string   `json:"desktopAppId,omitempty"`
	Busy         bool     `json:"busy,omitempty"`
	Windows      []Window `json:"windows"`
}

type Window struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	WorkspaceID string            `json:"workspaceId,omitempty"`
	MonitorRef  string            `json:"monitorRef,omitempty"`
	Focused     bool              `json:"focused,omitempty"`
	Minimized   bool              `json:"minimized,omitempty"`
	MRU         uint64            `json:"mru,omitempty"`
	Attention   surface.Attention `json:"attention,omitempty"`
}

func Decode(data []byte) (Snapshot, error) {
	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("decode GNOME Shell snapshot: %w", err)
	}
	if err := snapshot.Validate(); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func (s Snapshot) Validate() error {
	if s.Schema != SchemaVersion {
		return fmt.Errorf("unsupported GNOME Shell snapshot schema %d", s.Schema)
	}
	if s.Revision == 0 {
		return errors.New("GNOME Shell snapshot revision is required")
	}
	if s.CapturedAt.IsZero() {
		return errors.New("GNOME Shell capturedAt is required")
	}

	apps := map[string]struct{}{}
	windows := map[string]struct{}{}
	for _, app := range s.Apps {
		appID := strings.TrimSpace(app.AppID)
		if appID == "" {
			return errors.New("GNOME Shell application id is required")
		}
		if _, exists := apps[appID]; exists {
			return fmt.Errorf("duplicate GNOME Shell application %q", appID)
		}
		apps[appID] = struct{}{}
		for _, window := range app.Windows {
			if strings.TrimSpace(window.ID) == "" {
				return fmt.Errorf("GNOME Shell application %q contains a window without id", appID)
			}
			if _, exists := windows[window.ID]; exists {
				return fmt.Errorf("duplicate GNOME Shell window %q", window.ID)
			}
			windows[window.ID] = struct{}{}
			switch window.Attention {
			case "", surface.AttentionNormal, surface.AttentionWanted, surface.AttentionUrgent:
			default:
				return fmt.Errorf("GNOME Shell window %q has invalid attention %q", window.ID, window.Attention)
			}
		}
	}
	return nil
}

func (s Snapshot) ProviderSnapshots() []providers.Snapshot {
	out := make([]providers.Snapshot, 0, len(s.Apps))
	for _, app := range s.Apps {
		running := surface.LifecycleRunning
		focused := false
		attention := surface.AttentionNormal
		windows := make([]providers.WindowPatch, 0, len(app.Windows))
		for _, window := range app.Windows {
			focused = focused || window.Focused
			attention = strongestAttention(attention, window.Attention)
			windows = append(windows, providers.WindowPatch{
				WindowID:           surface.WindowID(window.ID),
				Title:              window.Title,
				WorkspaceID:        window.WorkspaceID,
				MonitorRef:         window.MonitorRef,
				Focused:            window.Focused,
				Minimized:          window.Minimized,
				MRU:                window.MRU,
				AuthoritativeState: true,
			})
		}
		out = append(out, providers.Snapshot{
			ProviderID:      ProviderID,
			Kind:            providers.SourceNative,
			AppID:           surface.ApplicationID(app.AppID),
			ObservedAt:      s.CapturedAt,
			TTL:             5 * time.Second,
			Priority:        200,
			Revision:        s.Revision,
			AllowOrphan:     true,
			ApplicationName: app.Name,
			IconName:        app.IconName,
			DesktopAppID:    app.DesktopAppID,
			Windows:         windows,
			Status: surface.StatusPatch{
				Lifecycle: &running,
				Focused:   &focused,
				Busy:      &app.Busy,
				Attention: &attention,
			},
			Capabilities: []surface.Capability{surface.CapabilityWindowObserve},
			Confidence:   surface.ConfidenceAuthoritative,
		})
	}
	return out
}

func strongestAttention(a, b surface.Attention) surface.Attention {
	if rank(b) > rank(a) {
		return b
	}
	return a
}

func rank(value surface.Attention) int {
	switch value {
	case surface.AttentionUrgent:
		return 3
	case surface.AttentionWanted:
		return 2
	default:
		return 1
	}
}
