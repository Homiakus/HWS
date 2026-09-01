package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type SurfaceLifecycle string
type SurfaceAttention string
type SurfaceActivity string
type SurfaceResourceState string
type SurfaceCapability string

const (
	SurfaceLifecycleUnknown		SurfaceLifecycle = "unknown"
	SurfaceLifecycleStopped		SurfaceLifecycle = "stopped"
	SurfaceLifecycleStarting	SurfaceLifecycle = "starting"
	SurfaceLifecycleRunning		SurfaceLifecycle = "running"
	SurfaceLifecycleSuspended	SurfaceLifecycle = "suspended"
	SurfaceLifecycleCrashed		SurfaceLifecycle = "crashed"
)

const (
	SurfaceAttentionUnknown	SurfaceAttention = "unknown"
	SurfaceAttentionNormal	SurfaceAttention = "normal"
	SurfaceAttentionNotice	SurfaceAttention = "notice"
	SurfaceAttentionUrgent	SurfaceAttention = "urgent"
)

const (
	SurfaceActivityUnknown	SurfaceActivity = "unknown"
	SurfaceActivityIdle	SurfaceActivity = "idle"
	SurfaceActivityWorking	SurfaceActivity = "working"
	SurfaceActivityWaiting	SurfaceActivity = "waiting"
	SurfaceActivityProgress	SurfaceActivity = "progress"
)

const (
	SurfaceResourceUnknown	SurfaceResourceState = "unknown"
	SurfaceResourceClean	SurfaceResourceState = "clean"
	SurfaceResourceDirty	SurfaceResourceState = "dirty"
	SurfaceResourceSyncing	SurfaceResourceState = "syncing"
	SurfaceResourceError	SurfaceResourceState = "error"
)

const (
	CapabilityWindowList		SurfaceCapability = "window.list"
	CapabilityWindowActivate	SurfaceCapability = "window.activate"
	CapabilityWindowPreview		SurfaceCapability = "window.preview"
	CapabilityViewList		SurfaceCapability = "view.list"
	CapabilityViewActivate		SurfaceCapability = "view.activate"
	CapabilityMediaObserve		SurfaceCapability = "media.observe"
)

type SurfaceObjectRef struct {
	ProviderID	SurfaceProviderID	`json:"provider_id"`
	SessionID	string			`json:"session_id"`
	LocalID		string			`json:"local_id"`
}

func (r SurfaceObjectRef) Valid() bool {
	return validID(r.ProviderID) && strings.TrimSpace(r.SessionID) != "" && strings.TrimSpace(r.LocalID) != ""
}

type SurfaceMediaState struct {
	Audio		bool	`json:"audio"`
	Microphone	bool	`json:"microphone"`
	Camera		bool	`json:"camera"`
}

type SurfaceWindow struct {
	ID		SurfaceWindowID		`json:"id"`
	Source		SurfaceObjectRef	`json:"source"`
	Title		string			`json:"title,omitempty"`
	Focused		bool			`json:"focused"`
	Workspace	int			`json:"workspace,omitempty"`
	Monitor		string			`json:"monitor,omitempty"`
	MRURank		int			`json:"mru_rank,omitempty"`
}

type SurfaceView struct {
	ID		SurfaceViewID		`json:"id"`
	Source		SurfaceObjectRef	`json:"source"`
	WindowID	SurfaceWindowID		`json:"window_id,omitempty"`
	Kind		string			`json:"kind,omitempty"`
	Title		string			`json:"title,omitempty"`
	ResourceRef	string			`json:"resource_ref,omitempty"`
	Active		bool			`json:"active"`
	Pinned		bool			`json:"pinned"`
	Dirty		bool			`json:"dirty"`
	ProgressKnown	bool			`json:"progress_known"`
	Progress	float64			`json:"progress,omitempty"`
	Attention	SurfaceAttention	`json:"attention,omitempty"`
	MRURank		int			`json:"mru_rank,omitempty"`
}

type ApplicationSurface struct {
	ID		SurfaceID		`json:"id"`
	AppID		string			`json:"app_id"`
	Title		string			`json:"title,omitempty"`
	IconName	string			`json:"icon_name,omitempty"`
	Lifecycle	SurfaceLifecycle	`json:"lifecycle"`
	Attention	SurfaceAttention	`json:"attention"`
	Activity	SurfaceActivity		`json:"activity"`
	Resource	SurfaceResourceState	`json:"resource"`
	Media		SurfaceMediaState	`json:"media"`
	Windows		[]SurfaceWindow		`json:"windows,omitempty"`
	Views		[]SurfaceView		`json:"views,omitempty"`
	Capabilities	[]SurfaceCapability	`json:"capabilities,omitempty"`
	Providers	[]SurfaceProviderID	`json:"providers,omitempty"`
	Partial		bool			`json:"partial"`
	Stale		bool			`json:"stale"`
}

type SurfaceProviderSnapshot struct {
	ProviderID	SurfaceProviderID	`json:"provider_id"`
	SessionID	string			`json:"session_id,omitempty"`
	Revision	uint64			`json:"revision"`
	Authority	int			`json:"authority"`
	Connected	bool			`json:"connected"`
	Stale		bool			`json:"stale"`
	Partial		bool			`json:"partial"`
	Capabilities	[]SurfaceCapability	`json:"capabilities,omitempty"`
	Surfaces	[]ApplicationSurface	`json:"surfaces,omitempty"`
}

type SurfaceProviderStatus struct {
	ProviderID	SurfaceProviderID	`json:"provider_id"`
	Revision	uint64			`json:"revision"`
	Authority	int			`json:"authority"`
	Connected	bool			`json:"connected"`
	Stale		bool			`json:"stale"`
	Partial		bool			`json:"partial"`
	Capabilities	[]SurfaceCapability	`json:"capabilities,omitempty"`
}

type SurfaceSnapshot struct {
	Generation	uint64				`json:"generation"`
	Revision	string				`json:"revision"`
	Providers	[]SurfaceProviderStatus		`json:"providers,omitempty"`
	Surfaces	[]ApplicationSurface		`json:"surfaces,omitempty"`
}

type SurfaceDiff struct {
	Added		[]SurfaceID	`json:"added,omitempty"`
	Removed		[]SurfaceID	`json:"removed,omitempty"`
	Changed		[]SurfaceID	`json:"changed,omitempty"`
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
	if !validLifecycle(s.Lifecycle) {
		return fmt.Errorf("surface: %q has invalid lifecycle %q", s.ID, s.Lifecycle)
	}
	if !validAttention(s.Attention) {
		return fmt.Errorf("surface: %q has invalid attention %q", s.ID, s.Attention)
	}
	if !validActivity(s.Activity) {
		return fmt.Errorf("surface: %q has invalid activity %q", s.ID, s.Activity)
	}
	if !validResourceState(s.Resource) {
		return fmt.Errorf("surface: %q has invalid resource state %q", s.ID, s.Resource)
	}
	if err := validateCapabilities(s.Capabilities); err != nil {
		return fmt.Errorf("surface: %q: %w", s.ID, err)
	}

	windowIDs := make(map[SurfaceWindowID]struct{}, len(s.Windows))
	for _, window := range s.Windows {
		if !validID(window.ID) || !window.Source.Valid() {
			return fmt.Errorf("surface: %q has invalid window identity", s.ID)
		}
		if window.MRURank < 0 {
			return fmt.Errorf("surface: %q window %q has negative MRU rank", s.ID, window.ID)
		}
		if _, ok := windowIDs[window.ID]; ok {
			return fmt.Errorf("surface: %q has duplicate window id %q", s.ID, window.ID)
		}
		windowIDs[window.ID] = struct{}{}
	}

	viewIDs := make(map[SurfaceViewID]struct{}, len(s.Views))
	for _, view := range s.Views {
		if !validID(view.ID) || !view.Source.Valid() {
			return fmt.Errorf("surface: %q has invalid view identity", s.ID)
		}
		if view.MRURank < 0 {
			return fmt.Errorf("surface: %q view %q has negative MRU rank", s.ID, view.ID)
		}
		if view.ProgressKnown && (view.Progress < 0 || view.Progress > 1) {
			return fmt.Errorf("surface: %q view %q progress must be between 0 and 1", s.ID, view.ID)
		}
		if view.Attention != "" && !validAttention(view.Attention) {
			return fmt.Errorf("surface: %q view %q has invalid attention %q", s.ID, view.ID, view.Attention)
		}
		if _, ok := viewIDs[view.ID]; ok {
			return fmt.Errorf("surface: %q has duplicate view id %q", s.ID, view.ID)
		}
		viewIDs[view.ID] = struct{}{}
	}
	return nil
}

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
			ProviderID:	provider.ProviderID,
			Revision:	provider.Revision,
			Authority:	provider.Authority,
			Connected:	provider.Connected,
			Stale:		provider.Stale,
			Partial:	provider.Partial,
			Capabilities:	normalizedCapabilities(provider.Capabilities),
		})
		if !provider.Connected {
			continue
		}

		for _, incoming := range provider.Surfaces {
			current := acc[incoming.ID]
			if current == nil {
				base := ApplicationSurface{
					ID:		incoming.ID,
					AppID:		incoming.AppID,
					Title:		incoming.Title,
					IconName:	incoming.IconName,
					Lifecycle:	normalizeLifecycle(incoming.Lifecycle),
					Attention:	normalizeAttention(incoming.Attention),
					Activity:	normalizeActivity(incoming.Activity),
					Resource:	normalizeResourceState(incoming.Resource),
					Media:		incoming.Media,
					Partial:	incoming.Partial || provider.Partial,
					Stale:		incoming.Stale || provider.Stale,
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
			for _, capability := range provider.Capabilities {
				capSeen[incoming.ID][capability] = struct{}{}
			}
			for _, capability := range incoming.Capabilities {
				capSeen[incoming.ID][capability] = struct{}{}
			}
			for _, window := range incoming.Windows {
				if _, exists := windowSeen[incoming.ID][window.ID]; exists {
					continue
				}
				windowSeen[incoming.ID][window.ID] = struct{}{}
				current.Windows = append(current.Windows, window)
			}
			for _, view := range incoming.Views {
				if _, exists := viewSeen[incoming.ID][view.ID]; exists {
					continue
				}
				viewSeen[incoming.ID][view.ID] = struct{}{}
				current.Views = append(current.Views, view)
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
			continue
		}
		if !sameSurface(oldSurface, newSurface) {
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

func (s SurfaceSnapshot) Clone() SurfaceSnapshot {
	out := SurfaceSnapshot{Generation: s.Generation, Revision: s.Revision}
	out.Providers = append([]SurfaceProviderStatus(nil), s.Providers...)
	for i := range out.Providers {
		out.Providers[i].Capabilities = append([]SurfaceCapability(nil), s.Providers[i].Capabilities...)
	}
	out.Surfaces = make([]ApplicationSurface, len(s.Surfaces))
	for i := range s.Surfaces {
		out.Surfaces[i] = cloneSurface(s.Surfaces[i])
	}
	return out
}

func cloneSurface(in ApplicationSurface) ApplicationSurface {
	out := in
	out.Windows = append([]SurfaceWindow(nil), in.Windows...)
	out.Views = append([]SurfaceView(nil), in.Views...)
	out.Capabilities = append([]SurfaceCapability(nil), in.Capabilities...)
	out.Providers = append([]SurfaceProviderID(nil), in.Providers...)
	return out
}

func normalizeSurface(surface *ApplicationSurface) {
	surface.Lifecycle = normalizeLifecycle(surface.Lifecycle)
	surface.Attention = normalizeAttention(surface.Attention)
	surface.Activity = normalizeActivity(surface.Activity)
	surface.Resource = normalizeResourceState(surface.Resource)
	sort.Slice(surface.Windows, func(i, j int) bool {
		if surface.Windows[i].MRURank != surface.Windows[j].MRURank {
			return surface.Windows[i].MRURank < surface.Windows[j].MRURank
		}
		return surface.Windows[i].ID < surface.Windows[j].ID
	})
	sort.Slice(surface.Views, func(i, j int) bool {
		if surface.Views[i].Active != surface.Views[j].Active {
			return surface.Views[i].Active
		}
		if surface.Views[i].Pinned != surface.Views[j].Pinned {
			return surface.Views[i].Pinned
		}
		if surface.Views[i].MRURank != surface.Views[j].MRURank {
			return surface.Views[i].MRURank < surface.Views[j].MRURank
		}
		return surface.Views[i].ID < surface.Views[j].ID
	})
	surface.Capabilities = normalizedCapabilities(surface.Capabilities)
	sort.Slice(surface.Providers, func(i, j int) bool { return surface.Providers[i] < surface.Providers[j] })
}

func mergeScalarSurface(dst *ApplicationSurface, src ApplicationSurface) {
	if dst.AppID == "" {
		dst.AppID = src.AppID
	}
	if dst.Title == "" {
		dst.Title = src.Title
	}
	if dst.IconName == "" {
		dst.IconName = src.IconName
	}
	if dst.Lifecycle == SurfaceLifecycleUnknown && src.Lifecycle != "" {
		dst.Lifecycle = normalizeLifecycle(src.Lifecycle)
	}
	if dst.Attention == SurfaceAttentionUnknown && src.Attention != "" {
		dst.Attention = normalizeAttention(src.Attention)
	}
	if dst.Activity == SurfaceActivityUnknown && src.Activity != "" {
		dst.Activity = normalizeActivity(src.Activity)
	}
	if dst.Resource == SurfaceResourceUnknown && src.Resource != "" {
		dst.Resource = normalizeResourceState(src.Resource)
	}
}

func surfaceSnapshotRevision(snapshot SurfaceSnapshot) (string, error) {
	type semanticProvider struct {
		ProviderID	SurfaceProviderID	`json:"provider_id"`
		Authority	int			`json:"authority"`
		Connected	bool			`json:"connected"`
		Stale		bool			`json:"stale"`
		Partial		bool			`json:"partial"`
		Capabilities	[]SurfaceCapability	`json:"capabilities,omitempty"`
	}
	type semanticSnapshot struct {
		Providers	[]semanticProvider	`json:"providers,omitempty"`
		Surfaces	[]ApplicationSurface	`json:"surfaces,omitempty"`
	}

	semantic := semanticSnapshot{Surfaces: snapshot.Surfaces}
	for _, provider := range snapshot.Providers {
		semantic.Providers = append(semantic.Providers, semanticProvider{
			ProviderID:	provider.ProviderID,
			Authority:	provider.Authority,
			Connected:	provider.Connected,
			Stale:		provider.Stale,
			Partial:	provider.Partial,
			Capabilities:	provider.Capabilities,
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

func normalizedCapabilities(in []SurfaceCapability) []SurfaceCapability {
	seen := make(map[SurfaceCapability]struct{}, len(in))
	out := make([]SurfaceCapability, 0, len(in))
	for _, capability := range in {
		if strings.TrimSpace(string(capability)) == "" {
			continue
		}
		if _, ok := seen[capability]; ok {
			continue
		}
		seen[capability] = struct{}{}
		out = append(out, capability)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func validateCapabilities(capabilities []SurfaceCapability) error {
	for _, capability := range capabilities {
		if strings.TrimSpace(string(capability)) == "" {
			return errors.New("empty capability")
		}
	}
	return nil
}

func normalizeLifecycle(state SurfaceLifecycle) SurfaceLifecycle {
	if state == "" {
		return SurfaceLifecycleUnknown
	}
	return state
}

func normalizeAttention(state SurfaceAttention) SurfaceAttention {
	if state == "" {
		return SurfaceAttentionUnknown
	}
	return state
}

func normalizeActivity(state SurfaceActivity) SurfaceActivity {
	if state == "" {
		return SurfaceActivityUnknown
	}
	return state
}

func normalizeResourceState(state SurfaceResourceState) SurfaceResourceState {
	if state == "" {
		return SurfaceResourceUnknown
	}
	return state
}

func validLifecycle(state SurfaceLifecycle) bool {
	switch normalizeLifecycle(state) {
	case SurfaceLifecycleUnknown, SurfaceLifecycleStopped, SurfaceLifecycleStarting, SurfaceLifecycleRunning, SurfaceLifecycleSuspended, SurfaceLifecycleCrashed:
		return true
	default:
		return false
	}
}

func validAttention(state SurfaceAttention) bool {
	switch normalizeAttention(state) {
	case SurfaceAttentionUnknown, SurfaceAttentionNormal, SurfaceAttentionNotice, SurfaceAttentionUrgent:
		return true
	default:
		return false
	}
}

func validActivity(state SurfaceActivity) bool {
	switch normalizeActivity(state) {
	case SurfaceActivityUnknown, SurfaceActivityIdle, SurfaceActivityWorking, SurfaceActivityWaiting, SurfaceActivityProgress:
		return true
	default:
		return false
	}
}

func validResourceState(state SurfaceResourceState) bool {
	switch normalizeResourceState(state) {
	case SurfaceResourceUnknown, SurfaceResourceClean, SurfaceResourceDirty, SurfaceResourceSyncing, SurfaceResourceError:
		return true
	default:
		return false
	}
}
