package browser

import (
	"fmt"
	"slices"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

type Tab struct {
	ID        int64  `json:"id"`
	WindowID  int64  `json:"windowId"`
	Title     string `json:"title"`
	Active    bool   `json:"active"`
	Pinned    bool   `json:"pinned"`
	Audible   bool   `json:"audible"`
	Muted     bool   `json:"muted"`
	Incognito bool   `json:"incognito"`
	MRU       uint64 `json:"mru"`
}
type Snapshot struct {
	Schema     uint32    `json:"schema"`
	Browser    string    `json:"browser"`
	AppID      string    `json:"appId"`
	Revision   uint64    `json:"revision"`
	CapturedAt time.Time `json:"capturedAt"`
	Tabs       []Tab     `json:"tabs"`
}

func (x Snapshot) ProviderSnapshot() providers.Snapshot {
	byWindow := map[int64][]surface.View{}
	for _, t := range x.Tabs {
		if t.Incognito {
			continue
		}
		meta := map[string]string{}
		if t.Audible {
			meta["audible"] = "true"
		}
		if t.Muted {
			meta["muted"] = "true"
		}
		byWindow[t.WindowID] = append(byWindow[t.WindowID], surface.View{ID: surface.ViewID(fmt.Sprintf("tab:%d", t.ID)), Kind: surface.ViewTab, Title: t.Title, Active: t.Active, Pinned: t.Pinned, MRU: t.MRU, Attention: surface.AttentionNormal, Metadata: meta})
	}
	windows := make([]providers.WindowPatch, 0, len(byWindow))
	for id, views := range byWindow {
		windows = append(windows, providers.WindowPatch{ProviderWindowID: fmt.Sprintf("%d", id), Views: views})
	}
	slices.SortFunc(windows, func(a, b providers.WindowPatch) int {
		if a.ProviderWindowID < b.ProviderWindowID {
			return -1
		}
		if a.ProviderWindowID > b.ProviderWindowID {
			return 1
		}
		return 0
	})
	return providers.Snapshot{ProviderID: "browser:" + x.Browser, Kind: providers.SourceExtension, AppID: surface.ApplicationID(x.AppID), IdentityHints: []string{x.Browser}, AllowOrphan: true, ApplicationName: x.Browser, ObservedAt: x.CapturedAt, TTL: 30 * time.Second, Priority: 100, Revision: x.Revision, Confidence: surface.ConfidenceAuthoritative, Windows: windows, Capabilities: []surface.Capability{surface.CapabilityViewObserve, surface.CapabilityViewActivate, surface.CapabilityViewClose}}
}
