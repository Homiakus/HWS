package launcherentry

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
	"github.com/godbus/dbus/v5"
)

type sink struct {
	snapshots []providers.Snapshot
}

func (s *sink) Ingest(snapshot providers.Snapshot) error {
	s.snapshots = append(s.snapshots, snapshot)
	return nil
}

func TestUpdateMergesPartialProperties(t *testing.T) {
	s := &sink{}
	collector := &Collector{sink: s, now: func() time.Time { return time.Unix(100, 0) }, entries: map[surface.ApplicationID]entryState{}}
	if err := collector.HandleSignal(&dbus.Signal{Name: interfaceName + "." + memberUpdate, Body: []any{
		"application://org.gnome.Nautilus.desktop",
		map[string]dbus.Variant{
			"count":         dbus.MakeVariant(int64(5)),
			"count-visible": dbus.MakeVariant(true),
		},
	}}); err != nil {
		t.Fatal(err)
	}
	if err := collector.HandleSignal(&dbus.Signal{Name: interfaceName + "." + memberUpdate, Body: []any{
		"application://org.gnome.Nautilus.desktop",
		map[string]dbus.Variant{
			"progress":         dbus.MakeVariant(0.42),
			"progress-visible": dbus.MakeVariant(true),
			"urgent":           dbus.MakeVariant(true),
		},
	}}); err != nil {
		t.Fatal(err)
	}
	last := s.snapshots[len(s.snapshots)-1]
	if last.Status.Badge == nil || *last.Status.Badge != "5" {
		t.Fatalf("badge=%v", last.Status.Badge)
	}
	if last.Status.Progress == nil || *last.Status.Progress != 0.42 {
		t.Fatalf("progress=%v", last.Status.Progress)
	}
	if last.Status.Attention == nil || *last.Status.Attention != surface.AttentionUrgent {
		t.Fatalf("attention=%v", last.Status.Attention)
	}
}

func TestVisibilityHidesCountAndProgress(t *testing.T) {
	state := entryState{AppID: "app.desktop", Count: 7, CountVisible: true, Progress: 0.5, ProgressVisible: true}
	applyProperties(&state, map[string]dbus.Variant{
		"count-visible":    dbus.MakeVariant(false),
		"progress-visible": dbus.MakeVariant(false),
	})
	if state.CountVisible || state.ProgressVisible {
		t.Fatalf("visibility not cleared: %#v", state)
	}
}

func TestAppIDFromURI(t *testing.T) {
	id, err := appIDFromURI("application://firefox.desktop")
	if err != nil || id != "firefox.desktop" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := appIDFromURI("file:///tmp/a.desktop"); err == nil {
		t.Fatal("non-application URI accepted")
	}
}
