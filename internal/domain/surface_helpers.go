package domain

import (
	"errors"
	"sort"
	"strings"
)

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
