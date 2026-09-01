package providers

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

func TestRegistryEnrichesSingleWindowAndReplacesProviderViews(t *testing.T) {
	now := time.Unix(100, 0)
	base := []surface.ApplicationSurface{{AppID: "firefox.desktop", Name: "Firefox", Windows: []surface.Window{{ID: "w1", Focused: true}}}}
	r := NewRegistry()
	progress := 0.4
	if err := r.Ingest(Snapshot{
		ProviderID: "browser", Kind: SourceExtension, AppID: "firefox.desktop", ObservedAt: now, TTL: time.Minute,
		Priority: 100, Revision: 1, Confidence: surface.ConfidenceHigh,
		Windows:      []WindowPatch{{ProviderWindowID: "7", Views: []surface.View{{ID: "t1", Kind: surface.ViewTab, Title: "HWS", Active: true}}}},
		Status:       surface.StatusPatch{Progress: &progress},
		Capabilities: []surface.Capability{surface.CapabilityViewObserve, surface.CapabilityViewActivate},
	}); err != nil {
		t.Fatal(err)
	}
	out := r.Apply(base, now)
	if len(out) != 1 || len(out[0].Windows[0].Views) != 1 {
		t.Fatalf("unexpected projection: %#v", out)
	}
	if out[0].Windows[0].Views[0].ProviderID != "browser" {
		t.Fatal("provider id not applied")
	}
	if out[0].ViewCount() != 1 || !out[0].Capabilities[surface.CapabilityViewActivate] {
		t.Fatal("capabilities/views missing")
	}

	if err := r.Ingest(Snapshot{ProviderID: "browser", Kind: SourceExtension, AppID: "firefox.desktop", ObservedAt: now.Add(time.Second), TTL: time.Minute, Priority: 100, Revision: 2, Confidence: surface.ConfidenceHigh, Windows: []WindowPatch{{ProviderWindowID: "7", Views: []surface.View{{ID: "t2", Kind: surface.ViewTab, Title: "Docs", Active: true}}}}}); err != nil {
		t.Fatal(err)
	}
	out = r.Apply(base, now.Add(time.Second))
	if got := out[0].Windows[0].Views[0].ID; got != "t2" {
		t.Fatalf("view=%q want=t2", got)
	}
}

func TestRegistrySkipsStaleAndRejectsRevisionRollback(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRegistry()
	s := Snapshot{ProviderID: "x", AppID: "app", ObservedAt: now.Add(-time.Minute), TTL: time.Second, Revision: 2}
	if err := r.Ingest(s); err != nil {
		t.Fatal(err)
	}
	out := r.Apply([]surface.ApplicationSurface{{AppID: "app", Name: "A"}}, now)
	if out[0].SourceRevision != nil && out[0].SourceRevision["x"] != 0 {
		t.Fatal("stale snapshot applied")
	}
	s.Revision = 1
	s.ObservedAt = now
	if err := r.Ingest(s); err == nil {
		t.Fatal("expected revision rollback error")
	}
}

func TestReplaceProviderIsAtomicAndRemovesMissingApplications(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRegistry()
	initial := []Snapshot{
		{ProviderID: "gnome-shell", Kind: SourceNative, AppID: "a.desktop", AllowOrphan: true, ObservedAt: now, Revision: 2},
		{ProviderID: "gnome-shell", Kind: SourceNative, AppID: "b.desktop", AllowOrphan: true, ObservedAt: now, Revision: 2},
	}
	if err := r.ReplaceProvider("gnome-shell", initial); err != nil {
		t.Fatal(err)
	}
	if err := r.ReplaceProvider("gnome-shell", []Snapshot{{
		ProviderID: "gnome-shell", Kind: SourceNative, AppID: "a.desktop", AllowOrphan: true, ObservedAt: now.Add(time.Second), Revision: 3,
	}}); err != nil {
		t.Fatal(err)
	}
	out := r.Apply(nil, now.Add(time.Second))
	if len(out) != 1 || out[0].AppID != "a.desktop" {
		t.Fatalf("provider replacement left stale applications: %#v", out)
	}

	if err := r.ReplaceProvider("gnome-shell", []Snapshot{{
		ProviderID: "gnome-shell", Kind: SourceNative, AppID: "a.desktop", AllowOrphan: true, ObservedAt: now.Add(2 * time.Second), Revision: 1,
	}}); err == nil {
		t.Fatal("stale replacement unexpectedly accepted")
	}
	out = r.Apply(nil, now.Add(time.Second))
	if len(out) != 1 || out[0].SourceRevision["gnome-shell"] != 3 {
		t.Fatal("failed replacement mutated registry")
	}
}

func TestProviderHealthDistinguishesFreshStaleAndPartial(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRegistry()
	if err := r.Ingest(Snapshot{
		ProviderID: "provider", Kind: SourceSystem, AppID: "fresh", ObservedAt: now, TTL: time.Minute, Revision: 4,
		Capabilities: []surface.Capability{surface.CapabilityMediaObserve},
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.Ingest(Snapshot{
		ProviderID: "provider", Kind: SourceSystem, AppID: "stale", ObservedAt: now.Add(-time.Minute), TTL: time.Second, Revision: 2,
	}); err != nil {
		t.Fatal(err)
	}
	health := r.Health(now)
	if len(health) != 1 || !health[0].Connected || !health[0].Partial || health[0].Stale {
		t.Fatalf("unexpected health: %#v", health)
	}
	if health[0].Revision != 4 || len(health[0].Capabilities) != 1 {
		t.Fatalf("health aggregation incomplete: %#v", health[0])
	}
}

func TestNativeIdentityHintsMergeRichProviderIntoCanonicalApp(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRegistry()
	if err := r.Ingest(Snapshot{
		ProviderID: "gnome-shell", Kind: SourceNative, AppID: "firefox_firefox.desktop",
		IdentityHints: []string{"firefox"}, AllowOrphan: true, ApplicationName: "Firefox", ObservedAt: now, TTL: time.Minute, Revision: 7,
		Windows: []WindowPatch{{WindowID: "window:1", Title: "Firefox", AuthoritativeState: true}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := r.Ingest(Snapshot{
		ProviderID: "browser:firefox", Kind: SourceExtension, AppID: "firefox.desktop",
		IdentityHints: []string{"firefox"}, AllowOrphan: true, ObservedAt: now, TTL: time.Minute, Revision: 2,
		Windows: []WindowPatch{{ProviderWindowID: "99", Views: []surface.View{{ID: "tab:1", Kind: surface.ViewTab, Title: "HWS"}}}},
	}); err != nil {
		t.Fatal(err)
	}
	out := r.Apply(nil, now)
	if len(out) != 1 {
		t.Fatalf("alias resolution produced %d applications: %#v", len(out), out)
	}
	app := out[0]
	if app.AppID != "firefox_firefox.desktop" || len(app.Windows) != 1 || app.Windows[0].ProviderOnly {
		t.Fatalf("rich provider did not merge into native window: %#v", app)
	}
	if len(app.Windows[0].Views) != 1 || app.Windows[0].Views[0].ID != "tab:1" {
		t.Fatalf("rich view missing after canonical merge: %#v", app.Windows[0].Views)
	}
}

func TestAmbiguousIdentityHintFailsClosed(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRegistry()
	for _, appID := range []surface.ApplicationID{"one.desktop", "two.desktop"} {
		if err := r.Ingest(Snapshot{
			ProviderID: "gnome-shell", Kind: SourceNative, AppID: appID, IdentityHints: []string{"shared"},
			AllowOrphan: true, ObservedAt: now, TTL: time.Minute, Revision: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.Ingest(Snapshot{
		ProviderID: "extension", Kind: SourceExtension, AppID: "shared.desktop", IdentityHints: []string{"shared"},
		AllowOrphan: true, ObservedAt: now, TTL: time.Minute, Revision: 1,
	}); err != nil {
		t.Fatal(err)
	}
	out := r.Apply(nil, now)
	if len(out) != 3 {
		t.Fatalf("ambiguous identity was incorrectly merged: %#v", out)
	}
}
