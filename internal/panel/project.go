package panel

import (
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

type Segment struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind,omitempty"`
	Title     string            `json:"title"`
	Active    bool              `json:"active"`
	Dirty     bool              `json:"dirty"`
	Attention surface.Attention `json:"attention"`
}

type AppCard struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	IconName       string            `json:"iconName,omitempty"`
	Title          string            `json:"title"`
	Subtitle       string            `json:"subtitle,omitempty"`
	Progress       *float64          `json:"progress,omitempty"`
	WindowCount    int               `json:"windowCount"`
	SurfaceCount   int               `json:"surfaceCount"`
	Focused        bool              `json:"focused"`
	Attention      surface.Attention `json:"attention"`
	Density        Density           `json:"density,omitempty"`
	MinWidth       int               `json:"minWidth,omitempty"`
	PreferredWidth int               `json:"preferredWidth,omitempty"`
	MaxWidth       int               `json:"maxWidth,omitempty"`
	Segments       []Segment         `json:"segments,omitempty"`
	OverflowCount  int               `json:"overflowCount,omitempty"`
}

type RenderConfig struct {
	Edge     Edge         `json:"edge"`
	Height   int          `json:"height"`
	Gap      int          `json:"gap"`
	Overflow OverflowMode `json:"overflow"`
}

type Snapshot struct {
	Revision    uint64       `json:"revision"`
	GeneratedAt time.Time    `json:"generatedAt"`
	Render      RenderConfig `json:"render"`
	Cards       []AppCard    `json:"cards"`
}

func Project(surfaces []surface.ApplicationSurface, maxSegments int, revision uint64, now time.Time) Snapshot {
	spec := DefaultSpec("runtime")
	spec.Groups = []Group{{
		ID: "applications",
		App: &AppConfig{
			Density:        DensityAdaptive,
			MinWidth:       64,
			PreferredWidth: 156,
			MaxWidth:       240,
			Surfaces:       SurfaceConfig{MaxVisible: maxSegments},
		},
	}}
	return ProjectWithSpec(surfaces, spec, revision, now)
}

func ProjectWithSpec(surfaces []surface.ApplicationSurface, spec Spec, revision uint64, now time.Time) Snapshot {
	appConfig := AppConfig{
		Density:        DensityAdaptive,
		MinWidth:       64,
		PreferredWidth: 156,
		MaxWidth:       240,
		Surfaces:       SurfaceConfig{MaxVisible: 4},
	}
	for _, group := range spec.Groups {
		if group.App != nil {
			appConfig = *group.App
			break
		}
	}
	maxSegments := appConfig.Surfaces.MaxVisible
	if maxSegments < 0 {
		maxSegments = 0
	}

	snap := Snapshot{
		Revision:    revision,
		GeneratedAt: now,
		Render: RenderConfig{
			Edge:     spec.Edge,
			Height:   spec.Height,
			Gap:      spec.Gap,
			Overflow: spec.Overflow,
		},
	}
	for _, a := range surfaces {
		card := AppCard{
			ID:             string(a.AppID),
			Name:           a.Name,
			IconName:       a.IconName,
			WindowCount:    a.WindowCount(),
			SurfaceCount:   a.ViewCount(),
			Focused:        a.Status.Focused,
			Attention:      a.Status.Attention,
			Progress:       a.Status.Progress,
			Subtitle:       a.Status.Summary,
			Density:        appConfig.Density,
			MinWidth:       appConfig.MinWidth,
			PreferredWidth: appConfig.PreferredWidth,
			MaxWidth:       appConfig.MaxWidth,
		}
		if v := a.ActiveView(); v != nil {
			card.Title = v.Title
		} else if focused := focusedWindow(a.Windows); focused != nil {
			card.Title = focused.Title
		} else if len(a.Windows) > 0 {
			card.Title = a.Windows[0].Title
		} else {
			card.Title = a.Name
		}

		all := projectedSegments(a)
		limit := len(all)
		if limit > maxSegments {
			limit = maxSegments
		}
		card.Segments = append(card.Segments, all[:limit]...)
		card.OverflowCount = len(all) - limit
		snap.Cards = append(snap.Cards, card)
	}
	return snap
}

func focusedWindow(windows []surface.Window) *surface.Window {
	for i := range windows {
		if windows[i].Focused {
			return &windows[i]
		}
	}
	return nil
}

func projectedSegments(app surface.ApplicationSurface) []Segment {
	var views []surface.View
	for _, window := range app.Windows {
		views = append(views, window.Views...)
	}
	if len(views) > 0 {
		out := make([]Segment, 0, len(views))
		for _, view := range views {
			out = append(out, Segment{
				ID:        string(view.ID),
				Kind:      "view",
				Title:     view.Title,
				Active:    view.Active,
				Dirty:     view.Dirty,
				Attention: view.Attention,
			})
		}
		return out
	}

	out := make([]Segment, 0, len(app.Windows))
	for _, window := range app.Windows {
		out = append(out, Segment{
			ID:        string(window.ID),
			Kind:      "window",
			Title:     window.Title,
			Active:    window.Focused,
			Attention: surface.AttentionNormal,
		})
	}
	return out
}
