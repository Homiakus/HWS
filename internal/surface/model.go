package surface

import (
	"cmp"
	"slices"
	"time"
)

type ApplicationID string

type WindowID string

type ViewID string

type Lifecycle string

const (
	LifecycleStopped   Lifecycle = "stopped"
	LifecycleStarting  Lifecycle = "starting"
	LifecycleRunning   Lifecycle = "running"
	LifecycleSuspended Lifecycle = "suspended"
	LifecycleStopping  Lifecycle = "stopping"
	LifecycleCrashed   Lifecycle = "crashed"
	LifecycleUnknown   Lifecycle = "unknown"
)

type Attention string

const (
	AttentionNormal Attention = "normal"
	AttentionWanted Attention = "attention"
	AttentionUrgent Attention = "urgent"
)

type Activity string

const (
	ActivityIdle     Activity = "idle"
	ActivityWorking  Activity = "working"
	ActivityProgress Activity = "progress"
	ActivityWaiting  Activity = "waiting"
	ActivityBlocked  Activity = "blocked"
)

type ResourceState string

const (
	ResourceClean    ResourceState = "clean"
	ResourceDirty    ResourceState = "dirty"
	ResourceSyncing  ResourceState = "syncing"
	ResourceConflict ResourceState = "conflict"
	ResourceError    ResourceState = "error"
	ResourceUnknown  ResourceState = "unknown"
)

type ViewKind string

const (
	ViewWindow   ViewKind = "window"
	ViewTab      ViewKind = "tab"
	ViewDocument ViewKind = "document"
	ViewEditor   ViewKind = "editor"
	ViewTerminal ViewKind = "terminal"
	ViewMedia    ViewKind = "media"
	ViewSession  ViewKind = "session"
	ViewCustom   ViewKind = "custom"
)

type Confidence uint8

const (
	ConfidenceLow Confidence = iota + 1
	ConfidenceMedium
	ConfidenceHigh
	ConfidenceAuthoritative
)

type Capability string

const (
	CapabilityWindowObserve  Capability = "window.observe"
	CapabilityWindowPreview  Capability = "window.preview"
	CapabilityWindowActivate Capability = "window.activate"
	CapabilityViewObserve    Capability = "view.observe"
	CapabilityViewActivate   Capability = "view.activate"
	CapabilityViewClose      Capability = "view.close"
	CapabilityViewReorder    Capability = "view.reorder"
	CapabilityViewProgress   Capability = "view.progress"
	CapabilityViewDirty      Capability = "view.dirty"
	CapabilityMediaObserve   Capability = "media.observe"
)

type AppStatus struct {
	Lifecycle  Lifecycle     `json:"lifecycle"`
	Focused    bool          `json:"focused"`
	Busy       bool          `json:"busy"`
	Attention  Attention     `json:"attention"`
	Activity   Activity      `json:"activity"`
	Resource   ResourceState `json:"resource"`
	Summary    string        `json:"summary,omitempty"`
	Detail     string        `json:"detail,omitempty"`
	Progress   *float64      `json:"progress,omitempty"`
	Badge      string        `json:"badge,omitempty"`
	UpdatedAt  time.Time     `json:"updatedAt"`
	Source     string        `json:"source,omitempty"`
	Confidence Confidence    `json:"confidence,omitempty"`
}

type StatusPatch struct {
	Lifecycle *Lifecycle
	Focused   *bool
	Busy      *bool
	Attention *Attention
	Activity  *Activity
	Resource  *ResourceState
	Summary   *string
	Detail    *string
	Progress  *float64
	Badge     *string
}

func (s *AppStatus) Apply(p StatusPatch, source string, confidence Confidence, at time.Time) {
	if p.Lifecycle != nil {
		s.Lifecycle = *p.Lifecycle
	}
	if p.Focused != nil {
		s.Focused = *p.Focused
	}
	if p.Busy != nil {
		s.Busy = *p.Busy
	}
	if p.Attention != nil {
		s.Attention = *p.Attention
	}
	if p.Activity != nil {
		s.Activity = *p.Activity
	}
	if p.Resource != nil {
		s.Resource = *p.Resource
	}
	if p.Summary != nil {
		s.Summary = *p.Summary
	}
	if p.Detail != nil {
		s.Detail = *p.Detail
	}
	if p.Progress != nil {
		v := *p.Progress
		if v < 0 {
			v = 0
		}
		if v > 1 {
			v = 1
		}
		s.Progress = &v
	}
	if p.Badge != nil {
		s.Badge = *p.Badge
	}
	s.Source = source
	s.Confidence = confidence
	s.UpdatedAt = at
}

type View struct {
	ID               ViewID            `json:"id"`
	ProviderID       string            `json:"providerId"`
	ProviderWindowID string            `json:"providerWindowId,omitempty"`
	Kind             ViewKind          `json:"kind"`
	Title            string            `json:"title"`
	Active           bool              `json:"active"`
	Dirty            bool              `json:"dirty"`
	Pinned           bool              `json:"pinned"`
	Attention        Attention         `json:"attention"`
	Progress         *float64          `json:"progress,omitempty"`
	ResourceRef      string            `json:"resourceRef,omitempty"`
	MRU              uint64            `json:"mru"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type Window struct {
	ID           WindowID `json:"id"`
	Title        string   `json:"title"`
	WorkspaceID  string   `json:"workspaceId,omitempty"`
	MonitorRef   string   `json:"monitorRef,omitempty"`
	Focused      bool     `json:"focused"`
	Minimized    bool     `json:"minimized"`
	MRU          uint64   `json:"mru"`
	ProviderOnly bool     `json:"providerOnly,omitempty"`
	Views        []View   `json:"views,omitempty"`
}

type ApplicationSurface struct {
	AppID          ApplicationID       `json:"appId"`
	Name           string              `json:"name"`
	IconName       string              `json:"iconName,omitempty"`
	DesktopAppID   string              `json:"desktopAppId,omitempty"`
	Windows        []Window            `json:"windows"`
	Status         AppStatus           `json:"status"`
	Capabilities   map[Capability]bool `json:"capabilities,omitempty"`
	SourceRevision map[string]uint64   `json:"sourceRevision,omitempty"`
}

func (a ApplicationSurface) Clone() ApplicationSurface {
	out := a
	out.Windows = make([]Window, len(a.Windows))
	for i := range a.Windows {
		out.Windows[i] = a.Windows[i]
		out.Windows[i].Views = append([]View(nil), a.Windows[i].Views...)
	}
	out.Capabilities = make(map[Capability]bool, len(a.Capabilities))
	for k, v := range a.Capabilities {
		out.Capabilities[k] = v
	}
	out.SourceRevision = make(map[string]uint64, len(a.SourceRevision))
	for k, v := range a.SourceRevision {
		out.SourceRevision[k] = v
	}
	return out
}

func (a *ApplicationSurface) Normalize() {
	for i := range a.Windows {
		slices.SortFunc(a.Windows[i].Views, func(x, y View) int {
			if x.Active != y.Active {
				if x.Active {
					return -1
				}
				return 1
			}
			if x.Attention != y.Attention {
				return cmp.Compare(attentionRank(y.Attention), attentionRank(x.Attention))
			}
			if x.Pinned != y.Pinned {
				if x.Pinned {
					return -1
				}
				return 1
			}
			if x.Dirty != y.Dirty {
				if x.Dirty {
					return -1
				}
				return 1
			}
			if x.MRU != y.MRU {
				return cmp.Compare(y.MRU, x.MRU)
			}
			return cmp.Compare(string(x.ID), string(y.ID))
		})
	}
	slices.SortFunc(a.Windows, func(x, y Window) int {
		if x.Focused != y.Focused {
			if x.Focused {
				return -1
			}
			return 1
		}
		if x.MRU != y.MRU {
			return cmp.Compare(y.MRU, x.MRU)
		}
		if x.ProviderOnly != y.ProviderOnly {
			if !x.ProviderOnly {
				return -1
			}
			return 1
		}
		return cmp.Compare(string(x.ID), string(y.ID))
	})
}

func (a ApplicationSurface) WindowCount() int { return len(a.Windows) }
func (a ApplicationSurface) ViewCount() int {
	n := 0
	for _, w := range a.Windows {
		n += len(w.Views)
	}
	return n
}

func (a ApplicationSurface) ActiveView() *View {
	for wi := range a.Windows {
		for vi := range a.Windows[wi].Views {
			if a.Windows[wi].Views[vi].Active {
				v := a.Windows[wi].Views[vi]
				return &v
			}
		}
	}
	return nil
}

func attentionRank(a Attention) int {
	switch a {
	case AttentionUrgent:
		return 3
	case AttentionWanted:
		return 2
	default:
		return 1
	}
}
