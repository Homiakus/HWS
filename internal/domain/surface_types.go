package domain

import (
	"errors"
	"fmt"
	"strings"
)

type SurfaceLifecycle string
type SurfaceAttention string
type SurfaceActivity string
type SurfaceResourceState string
type SurfaceCapability string

const (
	SurfaceLifecycleUnknown   SurfaceLifecycle = "unknown"
	SurfaceLifecycleStopped   SurfaceLifecycle = "stopped"
	SurfaceLifecycleStarting  SurfaceLifecycle = "starting"
	SurfaceLifecycleRunning   SurfaceLifecycle = "running"
	SurfaceLifecycleSuspended SurfaceLifecycle = "suspended"
	SurfaceLifecycleCrashed   SurfaceLifecycle = "crashed"
)

const (
	SurfaceAttentionUnknown SurfaceAttention = "unknown"
	SurfaceAttentionNormal  SurfaceAttention = "normal"
	SurfaceAttentionNotice  SurfaceAttention = "notice"
	SurfaceAttentionUrgent  SurfaceAttention = "urgent"
)

const (
	SurfaceActivityUnknown  SurfaceActivity = "unknown"
	SurfaceActivityIdle     SurfaceActivity = "idle"
	SurfaceActivityWorking  SurfaceActivity = "working"
	SurfaceActivityWaiting  SurfaceActivity = "waiting"
	SurfaceActivityProgress SurfaceActivity = "progress"
)

const (
	SurfaceResourceUnknown SurfaceResourceState = "unknown"
	SurfaceResourceClean   SurfaceResourceState = "clean"
	SurfaceResourceDirty   SurfaceResourceState = "dirty"
	SurfaceResourceSyncing SurfaceResourceState = "syncing"
	SurfaceResourceError   SurfaceResourceState = "error"
)

const (
	CapabilityWindowList     SurfaceCapability = "window.list"
	CapabilityWindowActivate SurfaceCapability = "window.activate"
	CapabilityWindowPreview  SurfaceCapability = "window.preview"
	CapabilityViewList       SurfaceCapability = "view.list"
	CapabilityViewActivate   SurfaceCapability = "view.activate"
	CapabilityMediaObserve   SurfaceCapability = "media.observe"
)

type SurfaceObjectRef struct {
	ProviderID SurfaceProviderID `json:"provider_id"`
	SessionID  string            `json:"session_id"`
	LocalID    string            `json:"local_id"`
}

func (r SurfaceObjectRef) Valid() bool {
	return validID(r.ProviderID) && strings.TrimSpace(r.SessionID) != "" && strings.TrimSpace(r.LocalID) != ""
}

type SurfaceMediaState struct {
	Audio      bool `json:"audio"`
	Microphone bool `json:"microphone"`
	Camera     bool `json:"camera"`
}

type SurfaceWindow struct {
	ID        SurfaceWindowID  `json:"id"`
	Source    SurfaceObjectRef `json:"source"`
	Title     string           `json:"title,omitempty"`
	Focused   bool             `json:"focused"`
	Workspace int              `json:"workspace,omitempty"`
	Monitor   string           `json:"monitor,omitempty"`
	MRURank   int              `json:"mru_rank,omitempty"`
}

type SurfaceView struct {
	ID            SurfaceViewID    `json:"id"`
	Source        SurfaceObjectRef `json:"source"`
	WindowID      SurfaceWindowID  `json:"window_id,omitempty"`
	Kind          string           `json:"kind,omitempty"`
	Title         string           `json:"title,omitempty"`
	ResourceRef   string           `json:"resource_ref,omitempty"`
	Active        bool             `json:"active"`
	Pinned        bool             `json:"pinned"`
	Dirty         bool             `json:"dirty"`
	ProgressKnown bool             `json:"progress_known"`
	Progress      float64          `json:"progress,omitempty"`
	Attention     SurfaceAttention `json:"attention,omitempty"`
	MRURank       int              `json:"mru_rank,omitempty"`
}

type ApplicationSurface struct {
	ID           SurfaceID            `json:"id"`
	AppID        string               `json:"app_id"`
	Title        string               `json:"title,omitempty"`
	IconName     string               `json:"icon_name,omitempty"`
	Lifecycle    SurfaceLifecycle     `json:"lifecycle"`
	Attention    SurfaceAttention     `json:"attention"`
	Activity     SurfaceActivity      `json:"activity"`
	Resource     SurfaceResourceState `json:"resource"`
	Media        SurfaceMediaState    `json:"media"`
	Windows      []SurfaceWindow      `json:"windows,omitempty"`
	Views        []SurfaceView        `json:"views,omitempty"`
	Capabilities []SurfaceCapability  `json:"capabilities,omitempty"`
	Providers    []SurfaceProviderID  `json:"providers,omitempty"`
	Partial      bool                 `json:"partial"`
	Stale        bool                 `json:"stale"`
}

type SurfaceProviderSnapshot struct {
	ProviderID   SurfaceProviderID    `json:"provider_id"`
	SessionID    string               `json:"session_id,omitempty"`
	Revision     uint64               `json:"revision"`
	Authority    int                  `json:"authority"`
	Connected    bool                 `json:"connected"`
	Stale        bool                 `json:"stale"`
	Partial      bool                 `json:"partial"`
	Capabilities []SurfaceCapability  `json:"capabilities,omitempty"`
	Surfaces     []ApplicationSurface `json:"surfaces,omitempty"`
}

type SurfaceProviderStatus struct {
	ProviderID   SurfaceProviderID   `json:"provider_id"`
	Revision     uint64              `json:"revision"`
	Authority    int                 `json:"authority"`
	Connected    bool                `json:"connected"`
	Stale        bool                `json:"stale"`
	Partial      bool                `json:"partial"`
	Capabilities []SurfaceCapability `json:"capabilities,omitempty"`
}

type SurfaceSnapshot struct {
	Generation uint64                  `json:"generation"`
	Revision   string                  `json:"revision"`
	Providers  []SurfaceProviderStatus `json:"providers,omitempty"`
	Surfaces   []ApplicationSurface    `json:"surfaces,omitempty"`
}

type SurfaceDiff struct {
	Added   []SurfaceID `json:"added,omitempty"`
	Removed []SurfaceID `json:"removed,omitempty"`
	Changed []SurfaceID `json:"changed,omitempty"`
}

func (p SurfaceProviderSnapshot) Validate() error {
	if !validID(p.ProviderID) {
		return errors.New("surface: provider id is required")
	}
	if (p.Connected || len(p.Surfaces) > 0) && strings.TrimSpace(p.SessionID) == "" {
		return fmt.Errorf("surface: connected provider %q requires session id", p.ProviderID)
	}
	if p.Authority < 0 {
		return fmt.Errorf("surface: provider %q authority must be non-negative", p.ProviderID)
	}
	if err := validateCapabilities(p.Capabilities); err != nil {
		return fmt.Errorf("surface: provider %q: %w", p.ProviderID, err)
	}
	seen := make(map[SurfaceID]struct{}, len(p.Surfaces))
	for _, surface := range p.Surfaces {
		if _, ok := seen[surface.ID]; ok {
			return fmt.Errorf("surface: provider %q has duplicate surface id %q", p.ProviderID, surface.ID)
		}
		seen[surface.ID] = struct{}{}
		if err := surface.Validate(); err != nil {
			return fmt.Errorf("surface: provider %q: %w", p.ProviderID, err)
		}
	}
	return nil
}

func (s ApplicationSurface) Validate() error {
	if !validID(s.ID) {
		return errors.New("surface: surface id is required")
	}
	if strings.TrimSpace(s.AppID) == "" {
		return fmt.Errorf("surface: app id is required for %q", s.ID)
	}
	if !validLifecycle(s.Lifecycle) || !validAttention(s.Attention) || !validActivity(s.Activity) || !validResourceState(s.Resource) {
		return fmt.Errorf("surface: %q has invalid state axes", s.ID)
	}
	if err := validateCapabilities(s.Capabilities); err != nil {
		return fmt.Errorf("surface: %q: %w", s.ID, err)
	}
	windowIDs := make(map[SurfaceWindowID]struct{}, len(s.Windows))
	for _, window := range s.Windows {
		if !validID(window.ID) || !window.Source.Valid() || window.MRURank < 0 {
			return fmt.Errorf("surface: %q has invalid window", s.ID)
		}
		if _, ok := windowIDs[window.ID]; ok {
			return fmt.Errorf("surface: %q has duplicate window id %q", s.ID, window.ID)
		}
		windowIDs[window.ID] = struct{}{}
	}
	viewIDs := make(map[SurfaceViewID]struct{}, len(s.Views))
	for _, view := range s.Views {
		if !validID(view.ID) || !view.Source.Valid() || view.MRURank < 0 {
			return fmt.Errorf("surface: %q has invalid view", s.ID)
		}
		if view.ProgressKnown && (view.Progress < 0 || view.Progress > 1) {
			return fmt.Errorf("surface: %q view %q progress must be between 0 and 1", s.ID, view.ID)
		}
		if view.Attention != "" && !validAttention(view.Attention) {
			return fmt.Errorf("surface: %q view %q has invalid attention", s.ID, view.ID)
		}
		if _, ok := viewIDs[view.ID]; ok {
			return fmt.Errorf("surface: %q has duplicate view id %q", s.ID, view.ID)
		}
		viewIDs[view.ID] = struct{}{}
	}
	return nil
}
