package vscode

import (
	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
	"time"
)

type Tab struct {
	ID       string `json:"id"`
	GroupID  string `json:"groupId"`
	Title    string `json:"title"`
	Resource string `json:"resource,omitempty"`
	Active   bool   `json:"active"`
	Dirty    bool   `json:"dirty"`
	Pinned   bool   `json:"pinned"`
	Preview  bool   `json:"preview"`
	Kind     string `json:"kind"`
	MRU      uint64 `json:"mru"`
}
type Snapshot struct {
	Schema     uint32    `json:"schema"`
	AppID      string    `json:"appId"`
	Revision   uint64    `json:"revision"`
	CapturedAt time.Time `json:"capturedAt"`
	Workspace  string    `json:"workspace"`
	Tabs       []Tab     `json:"tabs"`
}

func (x Snapshot) ProviderSnapshot() providers.Snapshot {
	views := make([]surface.View, 0, len(x.Tabs))
	dirty := false
	for _, t := range x.Tabs {
		kind := surface.ViewEditor
		if t.Kind == "terminal" {
			kind = surface.ViewTerminal
		}
		views = append(views, surface.View{ID: surface.ViewID(t.ID), Kind: kind, Title: t.Title, ResourceRef: t.Resource, Active: t.Active, Dirty: t.Dirty, Pinned: t.Pinned, MRU: t.MRU, Attention: surface.AttentionNormal})
		dirty = dirty || t.Dirty
	}
	rs := surface.ResourceClean
	if dirty {
		rs = surface.ResourceDirty
	}
	summary := x.Workspace
	return providers.Snapshot{ProviderID: "vscode", Kind: providers.SourceExtension, AppID: surface.ApplicationID(x.AppID), ObservedAt: x.CapturedAt, TTL: 30 * time.Second, Priority: 110, Revision: x.Revision, Confidence: surface.ConfidenceAuthoritative, Windows: []providers.WindowPatch{{Views: views}}, Status: surface.StatusPatch{Resource: &rs, Summary: &summary}, Capabilities: []surface.Capability{surface.CapabilityViewObserve, surface.CapabilityViewActivate, surface.CapabilityViewClose, surface.CapabilityViewDirty}}
}
