package gnomeshell

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/surface"
)

func TestSnapshotProjectsAuthoritativeWindows(t *testing.T) {
	at := time.Unix(100, 0).UTC()
	snapshot := Snapshot{
		Schema: SchemaVersion, Revision: 7, CapturedAt: at,
		Apps: []Application{{
			AppID: "org.gnome.Terminal.desktop", Name: "Terminal", Busy: true,
			Windows: []Window{
				{ID: "window:10", Title: "shell", WorkspaceID: "workspace:1", MonitorRef: "monitor:0", Focused: true, MRU: 44},
				{ID: "window:11", Title: "logs", Attention: surface.AttentionUrgent},
			},
		}},
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatal(err)
	}
	providers := snapshot.ProviderSnapshots()
	if len(providers) != 1 || len(providers[0].Windows) != 2 {
		t.Fatalf("unexpected provider projection: %#v", providers)
	}
	if !providers[0].Windows[0].AuthoritativeState || providers[0].Windows[0].WorkspaceID != "workspace:1" {
		t.Fatalf("window state was not preserved: %#v", providers[0].Windows[0])
	}
	if providers[0].Status.Focused == nil || !*providers[0].Status.Focused {
		t.Fatal("focused application state missing")
	}
	if providers[0].Status.Attention == nil || *providers[0].Status.Attention != surface.AttentionUrgent {
		t.Fatal("strongest attention was not projected")
	}
}

func TestSnapshotRejectsDuplicateWindowIdentity(t *testing.T) {
	snapshot := Snapshot{
		Schema: SchemaVersion, Revision: 1, CapturedAt: time.Now(),
		Apps: []Application{
			{AppID: "a", Windows: []Window{{ID: "window:1"}}},
			{AppID: "b", Windows: []Window{{ID: "window:1"}}},
		},
	}
	if err := snapshot.Validate(); err == nil {
		t.Fatal("duplicate window identity accepted")
	}
}
