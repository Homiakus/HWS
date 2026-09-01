package surface

import (
	"context"
	"errors"
	"testing"

	"github.com/Homiakus/HWS/internal/domain"
)

type fakeProvider struct {
	descriptor ProviderDescriptor
	snapshot   domain.SurfaceProviderSnapshot
	err        error
}

func (f fakeProvider) Descriptor() ProviderDescriptor {
	return f.descriptor
}

func (f fakeProvider) Snapshot(context.Context) (domain.SurfaceProviderSnapshot, error) {
	if f.err != nil {
		return domain.SurfaceProviderSnapshot{}, f.err
	}
	return f.snapshot, nil
}

func TestCollectorMergesProvidersAndKeepsFailureVisible(t *testing.T) {
	collector := NewCollector(
		fakeProvider{
			descriptor: ProviderDescriptor{ID: "gnome", Authority: 100, Capabilities: []domain.SurfaceCapability{domain.CapabilityWindowList}},
			snapshot: domain.SurfaceProviderSnapshot{
				SessionID: "shell-1",
				Connected: true,
				Surfaces: []domain.ApplicationSurface{{
					ID:        "app:zed",
					AppID:     "dev.zed.Zed.desktop",
					Title:     "Zed",
					Lifecycle: domain.SurfaceLifecycleRunning,
				}},
			},
		},
		fakeProvider{
			descriptor: ProviderDescriptor{ID: "zed-plugin", Authority: 80, Capabilities: []domain.SurfaceCapability{domain.CapabilityViewList}},
			err:        errors.New("plugin disconnected"),
		},
	)

	snapshot, failures, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Surfaces) != 1 || snapshot.Surfaces[0].Title != "Zed" {
		t.Fatalf("unexpected surfaces: %+v", snapshot.Surfaces)
	}
	if len(snapshot.Providers) != 2 {
		t.Fatalf("providers=%d want 2", len(snapshot.Providers))
	}
	if snapshot.Providers[1].ProviderID != "zed-plugin" || snapshot.Providers[1].Connected {
		t.Fatalf("failed provider not represented as disconnected: %+v", snapshot.Providers)
	}
	if len(failures) != 1 || failures[0].ProviderID != "zed-plugin" {
		t.Fatalf("failures=%+v", failures)
	}
}

func TestCollectorGenerationChangesOnlyWhenSemanticStateChanges(t *testing.T) {
	provider := &mutableFakeProvider{
		descriptor: ProviderDescriptor{ID: "gnome", Authority: 100},
		snapshot: domain.SurfaceProviderSnapshot{
			SessionID: "shell-1",
			Revision:  1,
			Connected: true,
			Surfaces: []domain.ApplicationSurface{{
				ID:        "app:terminal",
				AppID:     "org.gnome.Terminal.desktop",
				Lifecycle: domain.SurfaceLifecycleRunning,
			}},
		},
	}
	collector := NewCollector(provider)
	first, _, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	provider.snapshot.Revision++
	second, _, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Generation != second.Generation {
		t.Fatalf("generation changed for provider revision only: %d -> %d", first.Generation, second.Generation)
	}
	provider.snapshot.Surfaces[0].Attention = domain.SurfaceAttentionUrgent
	third, _, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if third.Generation != second.Generation+1 {
		t.Fatalf("generation=%d want %d", third.Generation, second.Generation+1)
	}
}

type mutableFakeProvider struct {
	descriptor ProviderDescriptor
	snapshot   domain.SurfaceProviderSnapshot
}

func (f *mutableFakeProvider) Descriptor() ProviderDescriptor {
	return f.descriptor
}

func (f *mutableFakeProvider) Snapshot(context.Context) (domain.SurfaceProviderSnapshot, error) {
	return f.snapshot, nil
}

func TestCollectorRejectsProviderIdentityMismatch(t *testing.T) {
	collector := NewCollector(fakeProvider{
		descriptor: ProviderDescriptor{ID: "gnome", Authority: 100},
		snapshot: domain.SurfaceProviderSnapshot{
			ProviderID: "not-gnome",
			SessionID:  "shell-1",
			Connected:  true,
		},
	})

	snapshot, failures, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) != 1 {
		t.Fatalf("failures=%v want 1", failures)
	}
	if len(snapshot.Surfaces) != 0 {
		t.Fatalf("unexpected surfaces: %+v", snapshot.Surfaces)
	}
	if len(snapshot.Providers) != 1 || snapshot.Providers[0].Connected {
		t.Fatalf("identity mismatch provider should remain visible as disconnected: %+v", snapshot.Providers)
	}
}
