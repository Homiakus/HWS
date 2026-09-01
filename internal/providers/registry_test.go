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
