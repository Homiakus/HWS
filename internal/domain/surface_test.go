package domain

import "testing"

func TestAggregateSurfaceSnapshotsDeterministicAndAuthorityAware(t *testing.T) {
	windowProvider := SurfaceProviderSnapshot{
		ProviderID:   "gnome",
		SessionID:    "shell-1",
		Revision:     10,
		Authority:    100,
		Connected:    true,
		Capabilities: []SurfaceCapability{CapabilityWindowList, CapabilityWindowActivate},
		Surfaces: []ApplicationSurface{{
			ID:        "app:firefox",
			AppID:     "firefox.desktop",
			Title:     "Firefox",
			Lifecycle: SurfaceLifecycleRunning,
			Windows: []SurfaceWindow{{
				ID:      "window:11",
				Source:  SurfaceObjectRef{ProviderID: "gnome", SessionID: "shell-1", LocalID: "11"},
				Focused: true,
			}},
		}},
	}
	viewProvider := SurfaceProviderSnapshot{
		ProviderID:   "firefox-extension",
		SessionID:    "browser-4",
		Revision:     77,
		Authority:    80,
		Connected:    true,
		Capabilities: []SurfaceCapability{CapabilityViewList, CapabilityViewActivate},
		Surfaces: []ApplicationSurface{{
			ID:        "app:firefox",
			AppID:     "firefox.desktop",
			Title:     "lower-authority-title",
			Lifecycle: SurfaceLifecycleUnknown,
			Views: []SurfaceView{{
				ID:       "view:github",
				Source:   SurfaceObjectRef{ProviderID: "firefox-extension", SessionID: "browser-4", LocalID: "tab-7"},
				WindowID: "window:11",
				Kind:     "tab",
				Title:    "Homiakus/HWS",
				Active:   true,
			}},
		}},
	}

	first, err := AggregateSurfaceSnapshots(SurfaceSnapshot{}, []SurfaceProviderSnapshot{viewProvider, windowProvider})
	if err != nil {
		t.Fatal(err)
	}
	second, err := AggregateSurfaceSnapshots(SurfaceSnapshot{}, []SurfaceProviderSnapshot{windowProvider, viewProvider})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != second.Revision {
		t.Fatalf("provider order changed revision: %s != %s", first.Revision, second.Revision)
	}
	if len(first.Surfaces) != 1 {
		t.Fatalf("surfaces=%d want 1", len(first.Surfaces))
	}
	surface := first.Surfaces[0]
	if surface.Title != "Firefox" {
		t.Fatalf("title=%q want Firefox", surface.Title)
	}
	if len(surface.Windows) != 1 || len(surface.Views) != 1 || len(surface.Capabilities) != 4 {
		t.Fatalf("unexpected merged surface: %+v", surface)
	}
}

func TestAggregateSurfaceSnapshotsStableGenerationForProviderRevisionOnly(t *testing.T) {
	provider := SurfaceProviderSnapshot{
		ProviderID: "gnome",
		SessionID:  "shell-1",
		Revision:   1,
		Authority:  100,
		Connected:  true,
		Surfaces:   []ApplicationSurface{{ID: "app:zed", AppID: "dev.zed.Zed.desktop", Lifecycle: SurfaceLifecycleRunning}},
	}
	first, err := AggregateSurfaceSnapshots(SurfaceSnapshot{}, []SurfaceProviderSnapshot{provider})
	if err != nil {
		t.Fatal(err)
	}
	provider.Revision++
	second, err := AggregateSurfaceSnapshots(first, []SurfaceProviderSnapshot{provider})
	if err != nil {
		t.Fatal(err)
	}
	if first.Revision != second.Revision || first.Generation != second.Generation {
		t.Fatalf("provider bookkeeping changed semantic generation: %d/%s -> %d/%s", first.Generation, first.Revision, second.Generation, second.Revision)
	}
}

func TestAggregateSurfaceSnapshotsDisconnectedProviderKeepsWindowFallback(t *testing.T) {
	connected := SurfaceProviderSnapshot{
		ProviderID: "gnome",
		SessionID:  "shell-1",
		Authority:  100,
		Connected:  true,
		Surfaces:   []ApplicationSurface{{ID: "app:firefox", AppID: "firefox.desktop", Lifecycle: SurfaceLifecycleRunning}},
	}
	disconnected := SurfaceProviderSnapshot{ProviderID: "firefox-extension", Authority: 80, Connected: false}
	snapshot, err := AggregateSurfaceSnapshots(SurfaceSnapshot{}, []SurfaceProviderSnapshot{disconnected, connected})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Surfaces) != 1 || len(snapshot.Providers) != 2 {
		t.Fatalf("unexpected fallback snapshot: %+v", snapshot)
	}
}

func TestAggregateSurfaceSnapshotsPropagatesPartialAndStale(t *testing.T) {
	provider := SurfaceProviderSnapshot{
		ProviderID: "tabs",
		SessionID:  "tabs-1",
		Authority:  10,
		Connected:  true,
		Stale:      true,
		Partial:    true,
		Surfaces:   []ApplicationSurface{{ID: "app:browser", AppID: "browser.desktop", Lifecycle: SurfaceLifecycleRunning}},
	}
	snapshot, err := AggregateSurfaceSnapshots(SurfaceSnapshot{}, []SurfaceProviderSnapshot{provider})
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Surfaces[0].Stale || !snapshot.Surfaces[0].Partial {
		t.Fatalf("surface stale/partial = %v/%v", snapshot.Surfaces[0].Stale, snapshot.Surfaces[0].Partial)
	}
}

func TestDiffSurfaceSnapshots(t *testing.T) {
	before := SurfaceSnapshot{Surfaces: []ApplicationSurface{
		{ID: "app:a", AppID: "a.desktop", Lifecycle: SurfaceLifecycleRunning},
		{ID: "app:b", AppID: "b.desktop", Lifecycle: SurfaceLifecycleRunning},
	}}
	after := SurfaceSnapshot{Surfaces: []ApplicationSurface{
		{ID: "app:b", AppID: "b.desktop", Lifecycle: SurfaceLifecycleCrashed},
		{ID: "app:c", AppID: "c.desktop", Lifecycle: SurfaceLifecycleRunning},
	}}
	diff := DiffSurfaceSnapshots(before, after)
	if len(diff.Added) != 1 || diff.Added[0] != "app:c" {
		t.Fatalf("added=%v", diff.Added)
	}
	if len(diff.Removed) != 1 || diff.Removed[0] != "app:a" {
		t.Fatalf("removed=%v", diff.Removed)
	}
	if len(diff.Changed) != 1 || diff.Changed[0] != "app:b" {
		t.Fatalf("changed=%v", diff.Changed)
	}
}

func TestSurfaceSnapshotCloneIsDeep(t *testing.T) {
	original := SurfaceSnapshot{Surfaces: []ApplicationSurface{{
		ID:           "app:a",
		AppID:        "a.desktop",
		Lifecycle:    SurfaceLifecycleRunning,
		Windows:      []SurfaceWindow{{ID: "window:1", Source: SurfaceObjectRef{ProviderID: "p", SessionID: "s", LocalID: "1"}}},
		Capabilities: []SurfaceCapability{CapabilityWindowList},
	}}}
	clone := original.Clone()
	clone.Surfaces[0].Windows[0].Title = "changed"
	clone.Surfaces[0].Capabilities[0] = CapabilityViewList
	if original.Surfaces[0].Windows[0].Title != "" || original.Surfaces[0].Capabilities[0] != CapabilityWindowList {
		t.Fatal("clone mutated original")
	}
}
