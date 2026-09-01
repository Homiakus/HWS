package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/Homiakus/HWS/internal/domain"
)

type Desktop struct {
	mu               sync.Mutex
	resources        map[domain.WorkspaceID]map[domain.ResourceID]domain.ResourceObservation
	ensureFailures   map[domain.ResourceID]int
	ensureAttempts   map[domain.ResourceID]int
	topologyRevision string
	observeFailure   error
}

func NewDesktop() *Desktop {
	return &Desktop{
		resources:        make(map[domain.WorkspaceID]map[domain.ResourceID]domain.ResourceObservation),
		ensureFailures:   make(map[domain.ResourceID]int),
		ensureAttempts:   make(map[domain.ResourceID]int),
		topologyRevision: "fake-topology-v1",
	}
}

func (d *Desktop) FailEnsure(id domain.ResourceID, attempts int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureFailures[id] = attempts
}

func (d *Desktop) FailObserve(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.observeFailure = err
}

func (d *Desktop) EnsureAttempts(id domain.ResourceID) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ensureAttempts[id]
}

func (d *Desktop) Seed(workspaceID domain.WorkspaceID, observation domain.ResourceObservation) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureWorkspace(workspaceID)[observation.ResourceID] = observation
}

func (d *Desktop) Observe(_ context.Context, desired domain.DesiredState) (domain.ObservedState, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.observeFailure != nil {
		return domain.ObservedState{}, d.observeFailure
	}
	copyResources := make(map[domain.ResourceID]domain.ResourceObservation)
	for id, observation := range d.ensureWorkspace(desired.WorkspaceID) {
		copyResources[id] = observation
	}
	return domain.ObservedState{
		WorkspaceID:      desired.WorkspaceID,
		TopologyRevision: d.topologyRevision,
		Resources:        copyResources,
	}, nil
}

func (d *Desktop) Ensure(_ context.Context, desired domain.DesiredState, resource domain.ResourceSpec) (domain.ResourceObservation, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureAttempts[resource.ID]++
	if remaining := d.ensureFailures[resource.ID]; remaining > 0 {
		d.ensureFailures[resource.ID] = remaining - 1
		return domain.ResourceObservation{}, fmt.Errorf("fake desktop: injected ensure failure for %s", resource.ID)
	}

	observation := domain.ResourceObservation{
		ResourceID: resource.ID,
		Present:    true,
		Ready:      true,
		Ownership:  resource.Ownership,
		SessionRef: fmt.Sprintf("fake:%s:%s", desired.WorkspaceID, resource.ID),
		AppID:      resource.DesktopAppID,
	}
	d.ensureWorkspace(desired.WorkspaceID)[resource.ID] = observation
	return observation, nil
}

func (d *Desktop) Close(_ context.Context, desired domain.DesiredState, resource domain.ResourceSpec, observed domain.ResourceObservation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if resource.Ownership != domain.OwnershipManaged || observed.Ownership != domain.OwnershipManaged {
		return nil
	}
	delete(d.ensureWorkspace(desired.WorkspaceID), resource.ID)
	return nil
}

func (d *Desktop) ensureWorkspace(id domain.WorkspaceID) map[domain.ResourceID]domain.ResourceObservation {
	resources := d.resources[id]
	if resources == nil {
		resources = make(map[domain.ResourceID]domain.ResourceObservation)
		d.resources[id] = resources
	}
	return resources
}
