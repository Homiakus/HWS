package surface

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/Homiakus/HWS/internal/domain"
)

type ProviderDescriptor struct {
	ID           domain.SurfaceProviderID
	Authority    int
	Capabilities []domain.SurfaceCapability
}

type Provider interface {
	Descriptor() ProviderDescriptor
	Snapshot(context.Context) (domain.SurfaceProviderSnapshot, error)
}

type ProviderFailure struct {
	ProviderID domain.SurfaceProviderID `json:"provider_id"`
	Error      string                   `json:"error"`
}

type Collector struct {
	mu        sync.Mutex
	providers []Provider
	current   domain.SurfaceSnapshot
}

func NewCollector(providers ...Provider) *Collector {
	return &Collector{providers: append([]Provider(nil), providers...)}
}

func (c *Collector) Collect(ctx context.Context) (domain.SurfaceSnapshot, []ProviderFailure, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	type result struct {
		snapshot domain.SurfaceProviderSnapshot
		failure  *ProviderFailure
	}

	results := make(chan result, len(c.providers))
	var wg sync.WaitGroup
	for _, provider := range c.providers {
		provider := provider
		wg.Add(1)
		go func() {
			defer wg.Done()
			descriptor := provider.Descriptor()
			if descriptor.ID == "" {
				results <- result{failure: &ProviderFailure{Error: "provider descriptor has empty id"}}
				return
			}
			if descriptor.Authority < 0 {
				results <- result{failure: &ProviderFailure{ProviderID: descriptor.ID, Error: "provider descriptor has negative authority"}}
				return
			}

			snapshot, err := provider.Snapshot(ctx)
			if err != nil {
				results <- result{
					snapshot: disconnectedSnapshot(descriptor, "provider-error"),
					failure:  &ProviderFailure{ProviderID: descriptor.ID, Error: err.Error()},
				}
				return
			}
			if snapshot.ProviderID != "" && snapshot.ProviderID != descriptor.ID {
				disconnected := disconnectedSnapshot(descriptor, snapshot.SessionID)
				results <- result{
					snapshot: disconnected,
					failure: &ProviderFailure{
						ProviderID: descriptor.ID,
						Error:      fmt.Sprintf("snapshot provider id %q does not match descriptor", snapshot.ProviderID),
					},
				}
				return
			}
			snapshot.ProviderID = descriptor.ID
			snapshot.Authority = descriptor.Authority
			snapshot.Capabilities = append(snapshot.Capabilities, descriptor.Capabilities...)
			results <- result{snapshot: snapshot}
		}()
	}
	wg.Wait()
	close(results)

	providerSnapshots := make([]domain.SurfaceProviderSnapshot, 0, len(c.providers))
	failures := make([]ProviderFailure, 0)
	for item := range results {
		if item.snapshot.ProviderID != "" {
			providerSnapshots = append(providerSnapshots, item.snapshot)
		}
		if item.failure != nil {
			failures = append(failures, *item.failure)
		}
	}
	sort.Slice(providerSnapshots, func(i, j int) bool { return providerSnapshots[i].ProviderID < providerSnapshots[j].ProviderID })
	sort.Slice(failures, func(i, j int) bool { return failures[i].ProviderID < failures[j].ProviderID })

	next, err := domain.AggregateSurfaceSnapshots(c.current, providerSnapshots)
	if err != nil {
		return c.current.Clone(), failures, err
	}
	c.current = next.Clone()
	return next.Clone(), failures, nil
}

func (c *Collector) Snapshot() domain.SurfaceSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current.Clone()
}

func disconnectedSnapshot(descriptor ProviderDescriptor, sessionID string) domain.SurfaceProviderSnapshot {
	if sessionID == "" {
		sessionID = "disconnected"
	}
	return domain.SurfaceProviderSnapshot{
		ProviderID:   descriptor.ID,
		SessionID:    sessionID,
		Authority:    descriptor.Authority,
		Connected:    false,
		Capabilities: append([]domain.SurfaceCapability(nil), descriptor.Capabilities...),
	}
}
