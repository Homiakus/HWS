package status

import (
	"fmt"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

type SNIItem struct {
	AppID    surface.ApplicationID
	Title    string
	Tooltip  string
	Status   string
	Revision uint64
}

func FromSNI(x SNIItem, now time.Time) providers.Snapshot {
	run := surface.LifecycleRunning
	att := surface.AttentionNormal
	activity := surface.ActivityIdle
	switch x.Status {
	case "NeedsAttention":
		att = surface.AttentionWanted
	case "Active":
		activity = surface.ActivityWorking
	}
	summary := x.Tooltip
	if summary == "" {
		summary = x.Title
	}
	return providers.Snapshot{ProviderID: "status-notifier", Kind: providers.SourceSystem, AppID: x.AppID, ObservedAt: now, TTL: 15 * time.Second, Priority: 30, Revision: x.Revision, Confidence: surface.ConfidenceHigh, Status: surface.StatusPatch{Lifecycle: &run, Attention: &att, Activity: &activity, Summary: &summary}}
}

type MPRIS struct {
	AppID          surface.ApplicationID
	PlaybackStatus string
	Title          string
	Artist         string
	PositionMicros int64
	LengthMicros   int64
	Revision       uint64
}

func FromMPRIS(x MPRIS, now time.Time) providers.Snapshot {
	run := surface.LifecycleRunning
	activity := surface.ActivityIdle
	if x.PlaybackStatus == "Playing" {
		activity = surface.ActivityWorking
	}
	summary := x.Title
	detail := x.Artist
	var progress *float64
	if x.LengthMicros > 0 {
		p := float64(x.PositionMicros) / float64(x.LengthMicros)
		progress = &p
	}
	return providers.Snapshot{ProviderID: "mpris", Kind: providers.SourceSystem, AppID: x.AppID, ObservedAt: now, TTL: 10 * time.Second, Priority: 40, Revision: x.Revision, Confidence: surface.ConfidenceHigh, Status: surface.StatusPatch{Lifecycle: &run, Activity: &activity, Summary: &summary, Detail: &detail, Progress: progress}, Capabilities: []surface.Capability{surface.CapabilityMediaObserve}}
}

type LauncherEntry struct {
	AppID    surface.ApplicationID
	Progress *float64
	Count    *int64
	Urgent   bool
	Updating bool
	Revision uint64
}

func FromLauncherEntry(x LauncherEntry, now time.Time) providers.Snapshot {
	run := surface.LifecycleRunning
	att := surface.AttentionNormal
	if x.Urgent {
		att = surface.AttentionUrgent
	}
	activity := surface.ActivityIdle
	if x.Progress != nil {
		activity = surface.ActivityProgress
	} else if x.Updating {
		activity = surface.ActivityWorking
	}
	var badge *string
	if x.Count != nil {
		s := fmt.Sprintf("%d", *x.Count)
		badge = &s
	}
	var summary *string
	if x.Updating {
		value := "Updating"
		summary = &value
	}
	return providers.Snapshot{ProviderID: "launcher-entry", Kind: providers.SourceSystem, AppID: x.AppID, ObservedAt: now, TTL: 30 * time.Second, Priority: 20, Revision: x.Revision, Confidence: surface.ConfidenceMedium, Status: surface.StatusPatch{Lifecycle: &run, Attention: &att, Activity: &activity, Progress: x.Progress, Badge: badge, Summary: summary}}
}
