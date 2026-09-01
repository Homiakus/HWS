package platform

import (
	"context"

	"github.com/Homiakus/HWS/internal/domain"
)

type Desktop interface {
	Observe(context.Context, domain.DesiredState) (domain.ObservedState, error)
	Ensure(context.Context, domain.DesiredState, domain.ResourceSpec) (domain.ResourceObservation, error)
	Close(context.Context, domain.DesiredState, domain.ResourceSpec, domain.ResourceObservation) error
}
