package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func AggregateSurfaceSnapshots(previous SurfaceSnapshot, providerSnapshots []SurfaceProviderSnapshot) (SurfaceSnapshot, error) {
	providers := append([]SurfaceProviderSnapshot(nil), providerSnapshots...)
	for i := range providers {
		if err := providers[i].Validate(); err != nil {
			return SurfaceSnapshot{}, err
		}
	}
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].Authority != providers[j].Authority {
			return providers[i].Authority > providers[j].Authority
		}
		return providers[i].ProviderID < providers[j].ProviderID
	})

	statuses := make([]SurfaceProviderStatus, 0, len(providers))
	acc := make(map[SurfaceID]*ApplicationSurface)
	windowSeen := make(map[SurfaceID]map[SurfaceWindowID]struct{})
	viewSeen := make(map[SurfaceID]map[SurfaceViewID]struct{})
	capSeen := make(map[SurfaceID]map[SurfaceCapability]struct{})
	providerSeen := make(map[SurfaceID]map[SurfaceProviderID]struct{})

	for _, provider := range providers {
		statuses = append(statuses, SurfaceProviderStatus{
			ProviderID:   provider.ProviderID,
			Revision:     provider.Revision,
			Authority:    provider.Authority,
			Connected:    provider.Connected,
			Stale:        provider.Stale,
			Partial:      provider.Partial,
			Capabilities: normalizedCapabilities(provider.Capabilities),
		})
		if !provider.Connected {
			continue
		}
		for _, incoming := range provider.Surfaces {
			current := acc[incoming.ID]
			if current == nil {
				base := ApplicationSurface{
					ID:        incoming.ID,
					AppID:     incoming.AppID,
					Title:     incoming.Title,
					IconName:  incoming.IconName,
					Lifecycle: normalizeLifecycle(incoming.Lifecycle),
					Attention: normalizeAttention(incoming.Attention),
					Activity:  normalizeActivity(incoming.Activity),
					Resource:  normalizeResourceState(incoming.Resource),
					Media:     incoming.Media,
					Partial:   incoming.Partial || provider.Partial,
					Stale:     incoming.Stale || provider.Stale,
				}
				acc[incoming.ID] = &base
				current = &base
				windowSeen[incoming.ID] = make(map[SurfaceWindowID]struct{})
				viewSeen[incoming.ID] = make(map[SurfaceViewID]struct{})
				capSeen[incoming.ID] = make(map[SurfaceCapability]struct{})
				providerSeen[incoming.ID] = make(map[SurfaceProviderID]struct{})
			} else {
				mergeScalarSurface(current, incoming)
				current.Media.Audio = current.Media.Audio || incoming.Media.Audio
				current.Media.Microphone = current.Media.Microphone || incoming.Media.Microphone
				current.Media.Camera = current.Media.Camera || incoming.Media.Camera
				current.Partial = current.Partial || incoming.Partial || provider.Partial
				current.Stale = current.Stale || incoming.Stale || provider.Stale
			}
			providerSeen[incoming.ID][provider.ProviderID] = struct{}{}
			for _, capability := range append(append([]SurfaceCapability(nil), provider.Capabilities...), incoming.Capabilities...) {
				if strings.TrimSpace(string(capability)) != "" {
					capSeen[incoming.ID][capability] = struct{}{}
				}
			}
			for _, window := range incoming.Windows {
				if _, exists := windowSeen[incoming.ID][window.ID]; !exists {
					windowSeen[incoming.ID][window.ID] = struct{}{}
					current.Windows = append(current.Windows, window)
				}
			}
			for _, view := range incoming.Views {
				if _, exists := viewSeen[incoming.ID][view.ID]; !exists {
					viewSeen[incoming.ID][view.ID] = struct{}{}
					current.Views = append(current.Views, view)
				}
			}
		}
	}

	surfaces := make([]ApplicationSurface, 0, len(acc))
	for id, surface := range acc {
		for capability := range capSeen[id] {
			surface.Capabilities = append(surface.Capabilities, capability)
		}
		for providerID := range providerSeen[id] {
			surface.Providers = append(surface.Providers, providerID)
		}
		normalizeSurface(surface)
		surfaces = append(surfaces, *surface)
	}
	sort.Slice(surfaces, func(i, j int) bool { return surfaces[i].ID < surfaces[j].ID })
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].ProviderID < statuses[j].ProviderID })

	next := SurfaceSnapshot{Providers: statuses, Surfaces: surfaces}
	revision, err := surfaceSnapshotRevision(next)
	if err != nil {
		return SurfaceSnapshot{}, err
	}
	next.Revision = revision
	if previous.Revision == revision {
		next.Generation = previous.Generation
	} else {
		next.Generation = previous.Generation + 1
	}
	return next, nil
}

func DiffSurfaceSnapshots(before, after SurfaceSnapshot) SurfaceDiff {
	left := make(map[SurfaceID]ApplicationSurface, len(before.Surfaces))
	right := make(map[SurfaceID]ApplicationSurface, len(after.Surfaces))
	for _, surface := range before.Surfaces {
		left[surface.ID] = surface
	}
	for _, surface := range after.Surfaces {
		right[surface.ID] = surface
	}
	var diff SurfaceDiff
	for id, oldSurface := range left {
		newSurface, ok := right[id]
		if !ok {
			diff.Removed = append(diff.Removed, id)
		} else if !sameSurface(oldSurface, newSurface) {
			diff.Changed = append(diff.Changed, id)
		}
	}
	for id := range right {
		if _, ok := left[id]; !ok {
			diff.Added = append(diff.Added, id)
		}
	}
	sort.Slice(diff.Added, func(i, j int) bool { return diff.Added[i] < diff.Added[j] })
	sort.Slice(diff.Removed, func(i, j int) bool { return diff.Removed[i] < diff.Removed[j] })
	sort.Slice(diff.Changed, func(i, j int) bool { return diff.Changed[i] < diff.Changed[j] })
	return diff
}

func surfaceSnapshotRevision(snapshot SurfaceSnapshot) (string, error) {
	type semanticProvider struct {
		ProviderID   SurfaceProviderID   `json:"provider_id"`
		Authority    int                 `json:"authority"`
		Connected    bool                `json:"connected"`
		Stale        bool                `json:"stale"`
		Partial      bool                `json:"partial"`
		Capabilities []SurfaceCapability `json:"capabilities,omitempty"`
	}
	type semanticSnapshot struct {
		Providers []semanticProvider   `json:"providers,omitempty"`
		Surfaces  []ApplicationSurface `json:"surfaces,omitempty"`
	}
	semantic := semanticSnapshot{Surfaces: snapshot.Surfaces}
	for _, provider := range snapshot.Providers {
		semantic.Providers = append(semantic.Providers, semanticProvider{
			ProviderID:   provider.ProviderID,
			Authority:    provider.Authority,
			Connected:    provider.Connected,
			Stale:        provider.Stale,
			Partial:      provider.Partial,
			Capabilities: provider.Capabilities,
		})
	}
	payload, err := json.Marshal(semantic)
	if err != nil {
		return "", fmt.Errorf("surface: encode snapshot revision: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func sameSurface(a, b ApplicationSurface) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return string(left) == string(right)
}
