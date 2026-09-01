package panel

import (
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

type Segment struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Active    bool              `json:"active"`
	Dirty     bool              `json:"dirty"`
	Attention surface.Attention `json:"attention"`
}
type AppCard struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	IconName      string            `json:"iconName,omitempty"`
	Title         string            `json:"title"`
	Subtitle      string            `json:"subtitle,omitempty"`
	Progress      *float64          `json:"progress,omitempty"`
	WindowCount   int               `json:"windowCount"`
	SurfaceCount  int               `json:"surfaceCount"`
	Focused       bool              `json:"focused"`
	Attention     surface.Attention `json:"attention"`
	Segments      []Segment         `json:"segments,omitempty"`
	OverflowCount int               `json:"overflowCount,omitempty"`
}
type Snapshot struct {
	Revision    uint64    `json:"revision"`
	GeneratedAt time.Time `json:"generatedAt"`
	Cards       []AppCard `json:"cards"`
}

func Project(surfaces []surface.ApplicationSurface, maxSegments int, revision uint64, now time.Time) Snapshot {
	if maxSegments < 0 {
		maxSegments = 0
	}
	snap := Snapshot{Revision: revision, GeneratedAt: now}
	for _, a := range surfaces {
		card := AppCard{ID: string(a.AppID), Name: a.Name, IconName: a.IconName, WindowCount: a.WindowCount(), SurfaceCount: a.ViewCount(), Focused: a.Status.Focused, Attention: a.Status.Attention, Progress: a.Status.Progress, Subtitle: a.Status.Summary}
		if v := a.ActiveView(); v != nil {
			card.Title = v.Title
		} else if len(a.Windows) > 0 {
			card.Title = a.Windows[0].Title
		} else {
			card.Title = a.Name
		}
		var all []surface.View
		for _, w := range a.Windows {
			all = append(all, w.Views...)
		}
		limit := len(all)
		if limit > maxSegments {
			limit = maxSegments
		}
		for i := 0; i < limit; i++ {
			v := all[i]
			card.Segments = append(card.Segments, Segment{ID: string(v.ID), Title: v.Title, Active: v.Active, Dirty: v.Dirty, Attention: v.Attention})
		}
		card.OverflowCount = len(all) - limit
		snap.Cards = append(snap.Cards, card)
	}
	return snap
}
