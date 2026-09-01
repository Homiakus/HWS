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
	WindowID         surface.WindowID `json:"windowId,omitempty"`
	ProviderWindowID string           `json:"providerWindowId,omitempty"`
	Title            string           `json:"title,omitempty"`
	Focused          bool             `json:"focused,omitempty"`
	Views            []surface.View   `json:"views,omitempty"`
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

func (r *Registry) Ingest(s Snapshot) error {
	if strings.TrimSpace(s.ProviderID) == "" {
		return errors.New("provider id is required")
	}
	if strings.TrimSpace(string(s.AppID)) == "" {
		return errors.New("app id is required")
	}
	if s.ObservedAt.IsZero() {
		return errors.New("observed time is required")
	}
	if s.TTL <= 0 {
		s.TTL = 5 * time.Second
	}
	if s.Confidence == 0 {
		s.Confidence = surface.ConfidenceMedium
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	key := snapshotKey(s)
	if prev, ok := r.snapshots[key]; ok && s.Revision < prev.Revision {
		return fmt.Errorf("stale provider revision: %d < %d", s.Revision, prev.Revision)
	}
	r.snapshots[key] = s
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

func (r *Registry) Apply(base []surface.ApplicationSurface, now time.Time) []surface.ApplicationSurface {
	byApp := make(map[surface.ApplicationID]*surface.ApplicationSurface, len(base))
	out := make([]surface.ApplicationSurface, 0, len(base))
	for _, item := range base {
		clone := item.Clone()
		out = append(out, clone)
		byApp[clone.AppID] = &out[len(out)-1]
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
		if now.Sub(s.ObservedAt) > s.TTL || s.ObservedAt.After(now.Add(time.Second)) {
			continue
		}
		app := byApp[s.AppID]
		if app == nil {
			if !s.AllowOrphan {
				continue
			}
			name := s.ApplicationName
			if name == "" {
				name = string(s.AppID)
			}
			out = append(out, surface.ApplicationSurface{AppID: s.AppID, Name: name})
			app = &out[len(out)-1]
			byApp[s.AppID] = app
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
			app.Windows = append(app.Windows, surface.Window{ID: id, Title: patch.Title, Focused: patch.Focused, ProviderOnly: true})
			idx = len(app.Windows) - 1
		}
		w := &app.Windows[idx]
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
