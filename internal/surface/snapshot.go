package surface

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

type ProviderHealth struct {
	ProviderID   string       `json:"providerId"`
	Kind         string       `json:"kind,omitempty"`
	Connected    bool         `json:"connected"`
	Stale        bool         `json:"stale"`
	Partial      bool         `json:"partial"`
	Revision     uint64       `json:"revision,omitempty"`
	Capabilities []Capability `json:"capabilities,omitempty"`
}

type Snapshot struct {
	Generation uint64               `json:"generation"`
	Revision   string               `json:"revision"`
	Surfaces   []ApplicationSurface `json:"surfaces"`
	Providers  []ProviderHealth     `json:"providers,omitempty"`
}

type SnapshotDiff struct {
	Added                []ApplicationID `json:"added,omitempty"`
	Removed              []ApplicationID `json:"removed,omitempty"`
	Changed              []ApplicationID `json:"changed,omitempty"`
	ProviderStateChanged bool            `json:"providerStateChanged,omitempty"`
}

func NewSnapshot(previous Snapshot, surfaces []ApplicationSurface, providers []ProviderHealth) (Snapshot, error) {
	next := Snapshot{
		Surfaces:  cloneAndNormalizeSurfaces(surfaces),
		Providers: normalizeProviderHealth(providers),
	}
	revision, err := semanticRevision(next)
	if err != nil {
		return Snapshot{}, err
	}
	next.Revision = revision
	if previous.Revision == revision {
		next.Generation = previous.Generation
	} else {
		next.Generation = previous.Generation + 1
	}
	return next, nil
}

func (s Snapshot) Clone() Snapshot {
	return Snapshot{
		Generation: s.Generation,
		Revision:   s.Revision,
		Surfaces:   cloneAndNormalizeSurfaces(s.Surfaces),
		Providers:  normalizeProviderHealth(s.Providers),
	}
}

func DiffSnapshots(before, after Snapshot) SnapshotDiff {
	left := make(map[ApplicationID]ApplicationSurface, len(before.Surfaces))
	right := make(map[ApplicationID]ApplicationSurface, len(after.Surfaces))
	for _, app := range before.Surfaces {
		left[app.AppID] = app
	}
	for _, app := range after.Surfaces {
		right[app.AppID] = app
	}
	var diff SnapshotDiff
	for id, oldApp := range left {
		newApp, ok := right[id]
		if !ok {
			diff.Removed = append(diff.Removed, id)
			continue
		}
		if !sameSemanticSurface(oldApp, newApp) {
			diff.Changed = append(diff.Changed, id)
		}
	}
	for id := range right {
		if _, ok := left[id]; !ok {
			diff.Added = append(diff.Added, id)
		}
	}
	slices.Sort(diff.Added)
	slices.Sort(diff.Removed)
	slices.Sort(diff.Changed)
	diff.ProviderStateChanged = !sameSemanticProviders(before.Providers, after.Providers)
	return diff
}

func cloneAndNormalizeSurfaces(in []ApplicationSurface) []ApplicationSurface {
	out := make([]ApplicationSurface, len(in))
	for i := range in {
		out[i] = deepCloneSurface(in[i])
		out[i].Normalize()
	}
	slices.SortFunc(out, func(a, b ApplicationSurface) int {
		if a.AppID < b.AppID {
			return -1
		}
		if a.AppID > b.AppID {
			return 1
		}
		return 0
	})
	return out
}

func deepCloneSurface(in ApplicationSurface) ApplicationSurface {
	out := in.Clone()
	if in.Status.Progress != nil {
		v := *in.Status.Progress
		out.Status.Progress = &v
	}
	for wi := range out.Windows {
		for vi := range out.Windows[wi].Views {
			source := in.Windows[wi].Views[vi]
			if source.Progress != nil {
				v := *source.Progress
				out.Windows[wi].Views[vi].Progress = &v
			}
			if source.Metadata != nil {
				out.Windows[wi].Views[vi].Metadata = make(map[string]string, len(source.Metadata))
				for k, v := range source.Metadata {
					out.Windows[wi].Views[vi].Metadata[k] = v
				}
			}
		}
	}
	return out
}

func normalizeProviderHealth(in []ProviderHealth) []ProviderHealth {
	out := make([]ProviderHealth, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Capabilities = append([]Capability(nil), in[i].Capabilities...)
		slices.Sort(out[i].Capabilities)
		out[i].Capabilities = slices.Compact(out[i].Capabilities)
	}
	slices.SortFunc(out, func(a, b ProviderHealth) int {
		if a.ProviderID < b.ProviderID {
			return -1
		}
		if a.ProviderID > b.ProviderID {
			return 1
		}
		if a.Kind < b.Kind {
			return -1
		}
		if a.Kind > b.Kind {
			return 1
		}
		return 0
	})
	return out
}

func semanticRevision(snapshot Snapshot) (string, error) {
	semanticSurfaces := make([]ApplicationSurface, len(snapshot.Surfaces))
	for i := range snapshot.Surfaces {
		semanticSurfaces[i] = deepCloneSurface(snapshot.Surfaces[i])
		semanticSurfaces[i].SourceRevision = nil
		semanticSurfaces[i].Status.UpdatedAt = time.Time{}
	}
	type semanticProvider struct {
		ProviderID   string       `json:"providerId"`
		Kind         string       `json:"kind,omitempty"`
		Connected    bool         `json:"connected"`
		Stale        bool         `json:"stale"`
		Partial      bool         `json:"partial"`
		Capabilities []Capability `json:"capabilities,omitempty"`
	}
	providers := make([]semanticProvider, len(snapshot.Providers))
	for i, p := range snapshot.Providers {
		providers[i] = semanticProvider{p.ProviderID, p.Kind, p.Connected, p.Stale, p.Partial, append([]Capability(nil), p.Capabilities...)}
	}
	payload, err := json.Marshal(struct {
		Surfaces  []ApplicationSurface `json:"surfaces"`
		Providers []semanticProvider   `json:"providers,omitempty"`
	}{semanticSurfaces, providers})
	if err != nil {
		return "", fmt.Errorf("surface snapshot: encode semantic state: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func sameSemanticSurface(a, b ApplicationSurface) bool {
	sa := Snapshot{Surfaces: []ApplicationSurface{a}}
	sb := Snapshot{Surfaces: []ApplicationSurface{b}}
	ra, _ := semanticRevision(sa)
	rb, _ := semanticRevision(sb)
	return ra == rb
}

func sameSemanticProviders(a, b []ProviderHealth) bool {
	sa := Snapshot{Providers: normalizeProviderHealth(a)}
	sb := Snapshot{Providers: normalizeProviderHealth(b)}
	ra, _ := semanticRevision(sa)
	rb, _ := semanticRevision(sb)
	return ra == rb
}
