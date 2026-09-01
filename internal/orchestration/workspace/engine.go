package workspace

import (
	"context"
	"fmt"

	"github.com/Homiakus/axiom"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/domain"
)

type Resolver interface {
	Resolve(domain.WorkspaceID, string) (domain.DesiredState, error)
}

type Lifecycle struct {
	engine *axiom.Engine
}

func Open(resolver Resolver, reconciler *reconcile.Reconciler) (*Lifecycle, error) {
	if resolver == nil {
		return nil, fmt.Errorf("workspace lifecycle: resolver is required")
	}
	if reconciler == nil {
		return nil, fmt.Errorf("workspace lifecycle: reconciler is required")
	}

	engine, err := axiom.Open(
		BuildDefinition(),
		axiom.ActTyped("ReconcileWorkspace", func(ctx context.Context, input ReconcileWorkspaceInput) (ReconcileWorkspaceOutput, error) {
			desired, err := resolver.Resolve(domain.WorkspaceID(input.WorkspaceID), input.DefinitionRevision)
			if err != nil {
				return ReconcileWorkspaceOutput{Status: StatusFailed, FailureCode: "definition_not_found"}, nil
			}
			result, err := reconciler.Reconcile(ctx, desired)
			if err != nil {
				return ReconcileWorkspaceOutput{}, err
			}
			return ReconcileWorkspaceOutput{
				Status:          result.TargetStatus(),
				ReachedRequired: result.Evaluation.RequiredReached,
				TotalRequired:   result.Evaluation.RequiredTotal,
				FailureCode:     result.FailureCode(),
			}, nil
		}),
		axiom.ActTyped("CloseWorkspace", func(ctx context.Context, input CloseWorkspaceInput) (CloseWorkspaceOutput, error) {
			desired, err := resolver.Resolve(domain.WorkspaceID(input.WorkspaceID), input.DefinitionRevision)
			if err != nil {
				return CloseWorkspaceOutput{Status: StatusFailed, FailureCode: "definition_not_found"}, nil
			}
			failures, err := reconciler.CloseManaged(ctx, desired)
			if err != nil {
				return CloseWorkspaceOutput{}, err
			}
			if len(failures) > 0 {
				return CloseWorkspaceOutput{Status: StatusDegraded, FailureCode: failures[0].Code}, nil
			}
			return CloseWorkspaceOutput{Status: StatusInactive}, nil
		}),
	)
	if err != nil {
		return nil, err
	}
	return &Lifecycle{engine: engine}, nil
}

func (l *Lifecycle) Activate(ctx context.Context, workspaceID domain.WorkspaceID, revision, operationKey string) error {
	if workspaceID == "" || revision == "" || operationKey == "" {
		return fmt.Errorf("workspace lifecycle: workspace id, revision and operation key are required")
	}
	return l.engine.Execution(string(workspaceID)).Dispatch(ctx, Activate{
		WorkspaceID:        string(workspaceID),
		DefinitionRevision: revision,
		OperationKey:       operationKey,
	})
}

func (l *Lifecycle) Recover(ctx context.Context, workspaceID domain.WorkspaceID, operationKey string) error {
	return l.engine.Execution(string(workspaceID)).Dispatch(ctx, Recover{OperationKey: operationKey})
}

func (l *Lifecycle) Close(ctx context.Context, workspaceID domain.WorkspaceID, operationKey string) error {
	return l.engine.Execution(string(workspaceID)).Dispatch(ctx, Close{OperationKey: operationKey})
}

func (l *Lifecycle) State(ctx context.Context, workspaceID domain.WorkspaceID) (State, error) {
	var state State
	if err := l.engine.Execution(string(workspaceID)).State(ctx, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (l *Lifecycle) Suspend(ctx context.Context, workspaceID domain.WorkspaceID) error {
	return l.engine.Execution(string(workspaceID)).Dispatch(ctx, Suspend{})
}

func (l *Lifecycle) Resume(ctx context.Context, workspaceID domain.WorkspaceID, operationKey string) error {
	if operationKey == "" {
		return fmt.Errorf("workspace lifecycle: operation key is required")
	}
	return l.engine.Execution(string(workspaceID)).Dispatch(ctx, Resume{OperationKey: operationKey})
}
