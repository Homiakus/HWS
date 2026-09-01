package surface

import (
	"testing"
	"time"
)

func TestSnapshotIsOrderIndependentAndIgnoresHeartbeatRevision(t *testing.T) {
	app := ApplicationSurface{AppID: "app:b", Name: "B", Status: AppStatus{Lifecycle: LifecycleRunning, UpdatedAt: time.Unix(10, 0)}, SourceRevision: map[string]uint64{"p": 1}}
	other := ApplicationSurface{AppID: "app:a", Name: "A", Status: AppStatus{Lifecycle: LifecycleRunning}}
	providers := []ProviderHealth{{ProviderID: "p", Connected: true, Revision: 1, Capabilities: []Capability{CapabilityViewObserve, CapabilityWindowObserve}}}
	first, err := NewSnapshot(Snapshot{}, []ApplicationSurface{app, other}, providers)
	if err != nil {
		t.Fatal(err)
	}

	app.Status.UpdatedAt = time.Unix(20, 0)
	app.SourceRevision["p"] = 2
	providers[0].Revision = 2
	providers[0].Capabilities = []Capability{CapabilityWindowObserve, CapabilityViewObserve}
	second, err := NewSnapshot(first, []ApplicationSurface{other, app}, providers)
	if err != nil {
		t.Fatal(err)
	}

	if first.Revision != second.Revision || first.Generation != second.Generation {
		t.Fatalf("heartbeat/order changed semantic snapshot: %d/%s -> %d/%s", first.Generation, first.Revision, second.Generation, second.Revision)
	}
	if second.Surfaces[0].AppID != "app:a" {
		t.Fatalf("surfaces not normalized: %+v", second.Surfaces)
	}
}

func TestSnapshotGenerationChangesForVisibleStateAndProviderHealth(t *testing.T) {
	app := ApplicationSurface{AppID: "app:a", Name: "A", Status: AppStatus{Lifecycle: LifecycleRunning}}
	first, _ := NewSnapshot(Snapshot{}, []ApplicationSurface{app}, []ProviderHealth{{ProviderID: "p", Connected: true}})
	app.Status.Attention = AttentionUrgent
	second, _ := NewSnapshot(first, []ApplicationSurface{app}, []ProviderHealth{{ProviderID: "p", Connected: true}})
	if second.Generation != first.Generation+1 {
		t.Fatalf("visible change generation=%d want %d", second.Generation, first.Generation+1)
	}
	third, _ := NewSnapshot(second, []ApplicationSurface{app}, []ProviderHealth{{ProviderID: "p", Connected: false}})
	if third.Generation != second.Generation+1 {
		t.Fatalf("provider health generation=%d want %d", third.Generation, second.Generation+1)
	}
}

func TestDiffSnapshots(t *testing.T) {
	before, _ := NewSnapshot(Snapshot{}, []ApplicationSurface{{AppID: "a", Name: "A"}, {AppID: "b", Name: "B"}}, []ProviderHealth{{ProviderID: "p", Connected: true}})
	after, _ := NewSnapshot(before, []ApplicationSurface{{AppID: "b", Name: "B2"}, {AppID: "c", Name: "C"}}, []ProviderHealth{{ProviderID: "p", Connected: false}})
	diff := DiffSnapshots(before, after)
	if len(diff.Added) != 1 || diff.Added[0] != "c" || len(diff.Removed) != 1 || diff.Removed[0] != "a" || len(diff.Changed) != 1 || diff.Changed[0] != "b" || !diff.ProviderStateChanged {
		t.Fatalf("unexpected diff: %+v", diff)
	}
}

func TestSnapshotCloneDeepCopiesProgressAndMetadata(t *testing.T) {
	progress := 0.5
	viewProgress := 0.25
	original, _ := NewSnapshot(Snapshot{}, []ApplicationSurface{{
		AppID: "a", Name: "A", Status: AppStatus{Progress: &progress}, Windows: []Window{{ID: "w", Views: []View{{ID: "v", Progress: &viewProgress, Metadata: map[string]string{"k": "v"}}}}},
	}}, nil)
	clone := original.Clone()
	*clone.Surfaces[0].Status.Progress = 0.9
	*clone.Surfaces[0].Windows[0].Views[0].Progress = 0.8
	clone.Surfaces[0].Windows[0].Views[0].Metadata["k"] = "changed"
	if *original.Surfaces[0].Status.Progress != 0.5 || *original.Surfaces[0].Windows[0].Views[0].Progress != 0.25 || original.Surfaces[0].Windows[0].Views[0].Metadata["k"] != "v" {
		t.Fatal("snapshot clone shares mutable nested state")
	}
}
