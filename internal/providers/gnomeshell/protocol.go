package gnomeshell

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
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
	Topology   Topology      `json:"topology,omitempty"`
	Apps       []Application `json:"apps"`
}

type Topology struct {
	Revision          string    `json:"revision"`
	PrimaryMonitorRef string    `json:"primaryMonitorRef"`
	Monitors          []Monitor `json:"monitors"`
}

type Monitor struct {
	Ref      string             `json:"ref"`
	Index    int                `json:"index"`
	Primary  bool               `json:"primary,omitempty"`
	Scale    float64            `json:"scale"`
	Geometry domain.LogicalRect `json:"geometry"`
	WorkArea domain.LogicalRect `json:"workArea"`
}

type Application struct {
	AppID         string   `json:"appId"`
	Name          string   `json:"name"`
	IconName      string   `json:"iconName,omitempty"`
	DesktopAppID  string   `json:"desktopAppId,omitempty"`
	IdentityHints []string `json:"identityHints,omitempty"`
	Busy          bool     `json:"busy,omitempty"`
	Windows       []Window `json:"windows"`
}

type Window struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	WorkspaceID string             `json:"workspaceId,omitempty"`
	MonitorRef  string             `json:"monitorRef,omitempty"`
	Frame       domain.LogicalRect `json:"frame,omitempty"`
	Focused     bool               `json:"focused,omitempty"`
	Minimized   bool               `json:"minimized,omitempty"`
	MRU         uint64             `json:"mru,omitempty"`
	Attention   surface.Attention  `json:"attention,omitempty"`
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
	if s.Topology.Revision != "" {
		if err := s.Topology.Validate(); err != nil {
			return err
		}
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
			if window.Frame != (domain.LogicalRect{}) && !window.Frame.Valid() {
				return fmt.Errorf("GNOME Shell window %q has invalid frame", window.ID)
			}
			switch window.Attention {
			case "", surface.AttentionNormal, surface.AttentionWanted, surface.AttentionUrgent:
			default:
				return fmt.Errorf("GNOME Shell window %q has invalid attention %q", window.ID, window.Attention)
			}
		}
	}
	return nil
}

func (t Topology) Validate() error {
	if strings.TrimSpace(t.Revision) == "" {
		return errors.New("GNOME Shell topology revision is required")
	}
	if len(t.Monitors) == 0 {
		return errors.New("GNOME Shell topology requires at least one monitor")
	}
	seen := make(map[string]struct{}, len(t.Monitors))
	primaryCount := 0
	for _, monitor := range t.Monitors {
		if strings.TrimSpace(monitor.Ref) == "" {
			return errors.New("GNOME Shell monitor ref is required")
		}
		if monitor.Index < 0 || monitor.Scale <= 0 || !monitor.Geometry.Valid() || !monitor.WorkArea.Valid() {
			return fmt.Errorf("GNOME Shell monitor %q is invalid", monitor.Ref)
		}
		if _, ok := seen[monitor.Ref]; ok {
			return fmt.Errorf("duplicate GNOME Shell monitor %q", monitor.Ref)
		}
		seen[monitor.Ref] = struct{}{}
		if monitor.Primary {
			primaryCount++
		}
	}
	if primaryCount != 1 {
		return fmt.Errorf("GNOME Shell topology requires exactly one primary monitor, got %d", primaryCount)
	}
	if _, ok := seen[t.PrimaryMonitorRef]; !ok {
		return fmt.Errorf("GNOME Shell primary monitor %q is unavailable", t.PrimaryMonitorRef)
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
			IdentityHints:   append([]string(nil), app.IdentityHints...),
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
