package panel

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

func TestProjectUsesNativeWindowsWhenNoRichViewsExist(t *testing.T) {
	spec := DefaultSpec("main")
	spec.Groups = []Group{{
		ID: "applications",
		App: &AppConfig{
			Density:        DensityCompact,
			MinWidth:       60,
			PreferredWidth: 120,
			MaxWidth:       180,
			Surfaces:       SurfaceConfig{MaxVisible: 2},
		},
	}}
	snapshot := ProjectWithSpec([]surface.ApplicationSurface{{
		AppID: "terminal.desktop",
		Name:  "Terminal",
		Windows: []surface.Window{
			{ID: "window:1", Title: "one", Focused: true},
			{ID: "window:2", Title: "two"},
		},
		Status: surface.AppStatus{Focused: true},
	}}, spec, 3, time.Unix(100, 0))

	card := snapshot.Cards[0]
	if len(card.Segments) != 2 || card.Segments[0].Kind != "window" || !card.Segments[0].Active {
		t.Fatalf("native window projection is wrong: %#v", card.Segments)
	}
	if card.Density != DensityCompact || card.MinWidth != 60 || snapshot.Render.Gap != spec.Gap {
		t.Fatalf("panel spec was not propagated: %#v %#v", card, snapshot.Render)
	}
}

func TestProjectPrefersRichViewsOverNativeWindowSegments(t *testing.T) {
	snapshot := Project([]surface.ApplicationSurface{{
		AppID: "code.desktop",
		Name:  "Code",
		Windows: []surface.Window{{
			ID: "window:1", Title: "Code",
			Views: []surface.View{{ID: "editor:1", Kind: surface.ViewEditor, Title: "main.go", Active: true}},
		}},
	}}, 4, 1, time.Unix(100, 0))
	if got := snapshot.Cards[0].Segments; len(got) != 1 || got[0].Kind != "view" || got[0].ID != "editor:1" {
		t.Fatalf("rich view projection is wrong: %#v", got)
	}
}
