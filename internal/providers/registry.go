package providers

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

type SourceKind string

const (
	SourceNative    SourceKind = "native"
	SourceExtension SourceKind = "extension"
	SourceATSPI     SourceKind = "atspi"
	SourceHeuristic SourceKind = "heuristic"
	SourceSystem    SourceKind = "system"
)

type WindowPatch struct {
	WindowID           surface.WindowID `json:"windowId,omitempty"`
	ProviderWindowID   string           `json:"providerWindowId,omitempty"`
	Title              string           `json:"title,omitempty"`
	WorkspaceID        string           `json:"workspaceId,omitempty"`
	MonitorRef         string           `json:"monitorRef,omitempty"`
	Focused            bool             `json:"focused,omitempty"`
	Minimized          bool             `json:"minimized,omitempty"`
	MRU                uint64           `json:"mru,omitempty"`
	AuthoritativeState bool             `json:"authoritativeState,omitempty"`
	Views              []surface.View   `json:"views,omitempty"`
}

type Snapshot struct {
	ProviderID      string                `json:"providerId"`
	Kind            SourceKind            `json:"kind"`
	AppID           surface.ApplicationID `json:"appId"`
	ObservedAt      time.Time             `json:"observedAt"`
	TTL             time.Duration         `json:"-"`
	Priority        int                   `json:"priority"`
	Revision        uint64                `json:"revision"`
	AllowOrphan     bool                  `json:"allowOrphan,omitempty"`
	ApplicationName string                `json:"applicationName,omitempty"`
	IconName        string                `json:"iconName,omitempty"`
	DesktopAppID    string                `json:"desktopAppId,omitempty"`
	Windows         []WindowPatch         `json:"windows,omitempty"`
	Status          surface.StatusPatch   `json:"-"`
	Capabilities    []surface.Capability  `json:"capabilities,omitempty"`
	Confidence      surface.Confidence    `json:"confidence"`
}

type Registry struct {
	mu        sync.RWMutex
	snapshots map[string]Snapshot
}

func NewRegistry() *Registry { return &Registry{snapshots: map[string]Snapshot{}} }

func snapshotKey(s Snapshot) string { return s.ProviderID + "\x00" + string(s.AppID) }

func normalizeSnapshot(s Snapshot) (Snapshot, error) {
	s.ProviderID = strings.TrimSpace(s.ProviderID)
	if s.ProviderID == "" {
		return Snapshot{}, errors.New("provider id is required")
	}
	if strings.TrimSpace(string(s.AppID)) == "" {
		return Snapshot{}, errors.New("app id is required")
	}
	if s.ObservedAt.IsZero() {
		return Snapshot{}, errors.New("observed time is required")
	}
	if s.TTL <= 0 {
		s.TTL = 5 * time.Second
	}
	if s.Confidence == 0 {
		s.Confidence = surface.ConfidenceMedium
	}
	return s, nil
}

func (r *Registry) Ingest(s Snapshot) error {
	normalized, err := normalizeSnapshot(s)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := snapshotKey(normalized)
	if prev, ok := r.snapshots[key]; ok && normalized.Revision < prev.Revision {
		return fmt.Errorf("stale provider revision: %d < %d", normalized.Revision, prev.Revision)
	}
	r.snapshots[key] = normalized
	return nil
}

// ReplaceProvider atomically replaces every application snapshot owned by one
// provider. It is intended for authoritative batch sources such as GNOME Shell
// where a single observation describes the provider's complete current world.
func (r *Registry) ReplaceProvider(providerID string, snapshots []Snapshot) error {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return errors.New("provider id is required")
	}

	next := make(map[string]Snapshot, len(snapshots))
	for _, raw := range snapshots {
		if strings.TrimSpace(raw.ProviderID) == "" {
			raw.ProviderID = providerID
		}
		if raw.ProviderID != providerID {
			return fmt.Errorf("provider replacement contains snapshot for %q", raw.ProviderID)
		}
		s, err := normalizeSnapshot(raw)
		if err != nil {
			return err
		}
		key := snapshotKey(s)
		if _, exists := next[key]; exists {
			return fmt.Errorf("duplicate application %q in provider replacement", s.AppID)
		}
		next[key] = s
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for key, s := range next {
		if prev, ok := r.snapshots[key]; ok && s.Revision < prev.Revision {
			return fmt.Errorf("stale provider revision: %d < %d", s.Revision, prev.Revision)
		}
	}
	for key, s := range r.snapshots {
		if s.ProviderID == providerID {
			delete(r.snapshots, key)
		}
	}
	for key, s := range next {
		r.snapshots[key] = s
	}
	return nil
}

func (r *Registry) DropProvider(providerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, s := range r.snapshots {
		if s.ProviderID == providerID {
			delete(r.snapshots, k)
		}
	}
}

func (r *Registry) Health(now time.Time) []surface.ProviderHealth {
	type aggregate struct {
		health     surface.ProviderHealth
		fresh      int
		stale      int
		capability map[surface.Capability]struct{}
	}
	byProvider := map[string]*aggregate{}

	r.mu.RLock()
	for _, s := range r.snapshots {
		a := byProvider[s.ProviderID]
		if a == nil {
			a = &aggregate{
				health: surface.ProviderHealth{
					ProviderID: s.ProviderID,
					Kind:       string(s.Kind),
				},
				capability: map[surface.Capability]struct{}{},
			}
			byProvider[s.ProviderID] = a
		}
		if s.Revision > a.health.Revision {
			a.health.Revision = s.Revision
		}
		for _, capability := range s.Capabilities {
			a.capability[capability] = struct{}{}
		}
		if snapshotFresh(s, now) {
			a.fresh++
		} else {
			a.stale++
		}
	}
	r.mu.RUnlock()

	out := make([]surface.ProviderHealth, 0, len(byProvider))
	for _, a := range byProvider {
		a.health.Connected = a.fresh > 0
		a.health.Stale = a.fresh == 0 && a.stale > 0
		a.health.Partial = a.fresh > 0 && a.stale > 0
		for capability := range a.capability {
			a.health.Capabilities = append(a.health.Capabilities, capability)
		}
		slices.Sort(a.health.Capabilities)
		out = append(out, a.health)
	}
	slices.SortFunc(out, func(a, b surface.ProviderHealth) int {
		if a.ProviderID < b.ProviderID {
			return -1
		}
		if a.ProviderID > b.ProviderID {
			return 1
		}
		return 0
	})
	return out
}

func snapshotFresh(s Snapshot, now time.Time) bool {
	return !s.ObservedAt.After(now.Add(time.Second)) && now.Sub(s.ObservedAt) <= s.TTL
}

func (r *Registry) Apply(base []surface.ApplicationSurface, now time.Time) []surface.ApplicationSurface {
	byApp := make(map[surface.ApplicationID]int, len(base))
	out := make([]surface.ApplicationSurface, 0, len(base))
	for _, item := range base {
		clone := item.Clone()
		out = append(out, clone)
		byApp[clone.AppID] = len(out) - 1
	}

	r.mu.RLock()
	snaps := make([]Snapshot, 0, len(r.snapshots))
	for _, s := range r.snapshots {
		snaps = append(snaps, s)
	}
	r.mu.RUnlock()
	slices.SortFunc(snaps, func(a, b Snapshot) int {
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		if a.Confidence != b.Confidence {
			return int(a.Confidence) - int(b.Confidence)
		}
		if a.ProviderID < b.ProviderID {
			return -1
		}
		if a.ProviderID > b.ProviderID {
			return 1
		}
		return 0
	})

	for _, s := range snaps {
		if !snapshotFresh(s, now) {
			continue
		}
		idx, ok := byApp[s.AppID]
		if !ok {
			if !s.AllowOrphan {
				continue
			}
			name := s.ApplicationName
			if name == "" {
				name = string(s.AppID)
			}
			out = append(out, surface.ApplicationSurface{
				AppID:        s.AppID,
				Name:         name,
				IconName:     s.IconName,
				DesktopAppID: s.DesktopAppID,
			})
			idx = len(out) - 1
			byApp[s.AppID] = idx
		}
		app := &out[idx]
		if app.Name == "" && s.ApplicationName != "" {
			app.Name = s.ApplicationName
		}
		if s.IconName != "" {
			app.IconName = s.IconName
		}
		if s.DesktopAppID != "" {
			app.DesktopAppID = s.DesktopAppID
		}
		if app.Capabilities == nil {
			app.Capabilities = map[surface.Capability]bool{}
		}
		if app.SourceRevision == nil {
			app.SourceRevision = map[string]uint64{}
		}
		for _, c := range s.Capabilities {
			app.Capabilities[c] = true
		}
		app.SourceRevision[s.ProviderID] = s.Revision
		app.Status.Apply(s.Status, s.ProviderID, s.Confidence, s.ObservedAt)
		mergeWindows(app, s)
	}
	for i := range out {
		out[i].Normalize()
	}
	return out
}

func mergeWindows(app *surface.ApplicationSurface, snap Snapshot) {
	for _, patch := range snap.Windows {
		idx := -1
		if patch.WindowID != "" {
			for i := range app.Windows {
				if app.Windows[i].ID == patch.WindowID {
					idx = i
					break
				}
			}
		}
		if idx < 0 && patch.WindowID == "" && len(app.Windows) == 1 {
			idx = 0
		}
		if idx < 0 {
			id := patch.WindowID
			if id == "" {
				id = surface.WindowID("provider:" + snap.ProviderID + ":" + patch.ProviderWindowID)
			}
			app.Windows = append(app.Windows, surface.Window{
				ID:           id,
				Title:        patch.Title,
				WorkspaceID:  patch.WorkspaceID,
				MonitorRef:   patch.MonitorRef,
				Focused:      patch.Focused,
				Minimized:    patch.Minimized,
				MRU:          patch.MRU,
				ProviderOnly: patch.WindowID == "",
			})
			idx = len(app.Windows) - 1
		}
		w := &app.Windows[idx]
		if patch.Title != "" {
			w.Title = patch.Title
		}
		if patch.AuthoritativeState {
			w.WorkspaceID = patch.WorkspaceID
			w.MonitorRef = patch.MonitorRef
			w.Focused = patch.Focused
			w.Minimized = patch.Minimized
			w.MRU = patch.MRU
			w.ProviderOnly = false
		}
		filtered := w.Views[:0]
		for _, v := range w.Views {
			if v.ProviderID != snap.ProviderID {
				filtered = append(filtered, v)
			}
		}
		w.Views = filtered
		for _, v := range patch.Views {
			v.ProviderID = snap.ProviderID
			if v.ProviderWindowID == "" {
				v.ProviderWindowID = patch.ProviderWindowID
			}
			w.Views = append(w.Views, v)
		}
	}
}
