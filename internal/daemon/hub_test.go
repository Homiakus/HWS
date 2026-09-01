package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

func TestHubUsesLastKnownGoodPanelAndProjectsRichViews(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "panel.hws.hcl")
	hub := NewHub(providers.NewRegistry())
	hub.now = func() time.Time { return time.Unix(100, 0) }
	if err := hub.Configure(path); err != nil {
		t.Fatal(err)
	}
	if err := hub.Ingest(providers.Snapshot{
		ProviderID: "browser:firefox", Kind: providers.SourceExtension, AppID: "firefox.desktop",
		AllowOrphan: true, ApplicationName: "Firefox", ObservedAt: time.Unix(100, 0), TTL: time.Minute,
		Revision: 1, Confidence: surface.ConfidenceAuthoritative,
		Capabilities: []surface.Capability{surface.CapabilityViewObserve, surface.CapabilityViewActivate},
		Windows:      []providers.WindowPatch{{ProviderWindowID: "1", Views: []surface.View{{ID: "tab:7", Kind: surface.ViewTab, Title: "HWS", Active: true}}}},
	}); err != nil {
		t.Fatal(err)
	}
	json, err := hub.PanelJSON()
	if err != nil {
		t.Fatal(err)
	}
	if json == "" || json == "{}" {
		t.Fatalf("empty panel projection: %q", json)
	}

	if err := os.WriteFile(path, []byte(`panel "broken" { height = 1 }`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := hub.ReloadPanel(); err == nil {
		t.Fatal("invalid config unexpectedly accepted")
	}
	if _, _, valid := hub.manager.Current(); !valid {
		t.Fatal("last-known-good panel was lost")
	}
}

func TestMaxVisibleSegmentsUsesPanelSpec(t *testing.T) {
	hub := NewHub(providers.NewRegistry())
	hub.now = func() time.Time { return time.Unix(100, 0) }
	path := filepath.Join(t.TempDir(), "panel.hws.hcl")
	src := `panel "main" {
  edge = "top"
  height = 40
  gap = 4
  overflow = "popover"
  group "applications" {
    app {
      density = "adaptive"
      min_width = 64
      preferred_width = 120
      max_width = 220
      surfaces { mode = "segments" max_visible = 1 overflow = "count" }
    }
  }
}`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := hub.Configure(path); err != nil {
		t.Fatal(err)
	}
	if err := hub.Ingest(providers.Snapshot{
		ProviderID: "vscode", Kind: providers.SourceExtension, AppID: "code.desktop", AllowOrphan: true,
		ApplicationName: "Code", ObservedAt: time.Unix(100, 0), TTL: time.Minute, Revision: 1,
		Windows: []providers.WindowPatch{{Views: []surface.View{{ID: "a", Title: "A"}, {ID: "b", Title: "B"}}}},
	}); err != nil {
		t.Fatal(err)
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if got := len(hub.panelSnapshot.Cards[0].Segments); got != 1 {
		t.Fatalf("segments=%d want 1", got)
	}
	if hub.panelSnapshot.Cards[0].OverflowCount != 1 {
		t.Fatalf("overflow=%d want 1", hub.panelSnapshot.Cards[0].OverflowCount)
	}
}
