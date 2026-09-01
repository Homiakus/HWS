package mpris

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/surface"
	"github.com/godbus/dbus/v5"
)

func TestSnapshotFromProperties(t *testing.T) {
	now := time.Unix(100, 0)
	root := map[string]dbus.Variant{
		"Identity":     dbus.MakeVariant("Music"),
		"DesktopEntry": dbus.MakeVariant("org.gnome.Music"),
	}
	player := map[string]dbus.Variant{
		"PlaybackStatus": dbus.MakeVariant("Playing"),
		"Position":       dbus.MakeVariant(int64(25)),
		"Metadata": dbus.MakeVariant(map[string]dbus.Variant{
			"xesam:title":  dbus.MakeVariant("Track"),
			"xesam:artist": dbus.MakeVariant([]string{"Artist A", "Artist B"}),
			"mpris:length": dbus.MakeVariant(int64(100)),
		}),
	}
	snapshot := snapshotFromProperties("org.mpris.MediaPlayer2.gnomemusic", root, player, 7, now)
	if snapshot.AppID != "org.gnome.Music.desktop" {
		t.Fatalf("appID=%q", snapshot.AppID)
	}
	if snapshot.ProviderID != "mpris:org.mpris.MediaPlayer2.gnomemusic" {
		t.Fatalf("provider=%q", snapshot.ProviderID)
	}
	if snapshot.Status.Activity == nil || *snapshot.Status.Activity != surface.ActivityWorking {
		t.Fatalf("activity=%v", snapshot.Status.Activity)
	}
	if snapshot.Status.Summary == nil || *snapshot.Status.Summary != "Track" {
		t.Fatalf("summary=%v", snapshot.Status.Summary)
	}
	if snapshot.Status.Detail == nil || *snapshot.Status.Detail != "Artist A, Artist B" {
		t.Fatalf("detail=%v", snapshot.Status.Detail)
	}
	if snapshot.Status.Progress == nil || *snapshot.Status.Progress != 0.25 {
		t.Fatalf("progress=%v", snapshot.Status.Progress)
	}
}

func TestFallbackDesktopEntryDropsInstanceSuffix(t *testing.T) {
	if got := fallbackDesktopEntry("org.mpris.MediaPlayer2.vlc.instance42"); got != "vlc" {
		t.Fatalf("fallback=%q", got)
	}
}
