package reconcile

import (
	"context"
	"fmt"

	"github.com/Homiakus/HWS/internal/domain"
	"github.com/Homiakus/HWS/internal/platform"
)

type Failure struct {
	ResourceID domain.ResourceID
	Code       string
	Err        error
}

type Result struct {
	Observed   domain.ObservedState
	Evaluation domain.ReconcileEvaluation
	Failures   []Failure
}

func (r Result) TargetStatus() string {
	if r.Evaluation.RequiredTotal == 0 || r.Evaluation.RequiredReached == r.Evaluation.RequiredTotal {
		return "active"
	}
	if r.Evaluation.RequiredReached > 0 {
		return "degraded"
	}
	return "failed"
}

func (r Result) FailureCode() string {
	if len(r.Failures) > 0 {
		return r.Failures[0].Code
	}
	if len(r.Evaluation.MissingRequired) > 0 {
		return "required_resource_unreached"
	}
	return ""
}

type Reconciler struct {
	desktop platform.Desktop
}

func New(desktop platform.Desktop) *Reconciler {
	return &Reconciler{desktop: desktop}
}

func (r *Reconciler) Reconcile(ctx context.Context, desired domain.DesiredState) (Result, error) {
	if err := desired.Validate(); err != nil {
		return Result{}, err
	}
	before, err := r.desktop.Observe(ctx, desired)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile: observe before actions: %w", err)
	}

	plan := domain.BuildReconcilePlan(desired, before)
	result := Result{}
	for _, action := range plan.Actions {
		switch action.Kind {
		case domain.ActionEnsure:
			if _, err := r.desktop.Ensure(ctx, desired, action.Resource); err != nil {
				result.Failures = append(result.Failures, Failure{
					ResourceID: action.Resource.ID,
					Code:       "ensure_failed",
					Err:        err,
				})
			}
		}
	}

	after, err := r.desktop.Observe(ctx, desired)
	if err != nil {
		return Result{}, fmt.Errorf("reconcile: observe after actions: %w", err)
	}
	result.Observed = after
	result.Evaluation = domain.EvaluateReconcile(desired, after)
	return result, nil
}

func (r *Reconciler) CloseManaged(ctx context.Context, desired domain.DesiredState) ([]Failure, error) {
	if err := desired.Validate(); err != nil {
		return nil, err
	}
	observed, err := r.desktop.Observe(ctx, desired)
	if err != nil {
		return nil, fmt.Errorf("close workspace: observe: %w", err)
	}

	var failures []Failure
	for _, resource := range desired.Resources {
		if resource.Ownership != domain.OwnershipManaged {
			continue
		}
		observation, ok := observed.Resources[resource.ID]
		if !ok || !observation.Present {
			continue
		}
		if err := r.desktop.Close(ctx, desired, resource, observation); err != nil {
			failures = append(failures, Failure{ResourceID: resource.ID, Code: "close_failed", Err: err})
		}
	}
	return failures, nil
}
