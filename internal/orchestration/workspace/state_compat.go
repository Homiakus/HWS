package workspace

import (
	"context"

	"github.com/Homiakus/HWS/internal/domain"
)

// StateOrInactive reads durable state without forcing creation of an Axiom
// execution. The pinned pre-v1 Axiom runtime currently exposes its
// ErrExecutionNotFound sentinel only from an internal package, so external
// consumers cannot use errors.Is against it. Keep the compatibility check
// isolated here at the Axiom boundary; do not copy it into daemon/UI code.
//
// When Axiom exposes a public classifier this exact-message shim must be
// replaced and the pinned compatibility test updated.
func (l *Lifecycle) StateOrInactive(ctx context.Context, workspaceID domain.WorkspaceID) (State, bool, error) {
	state, err := l.State(ctx, workspaceID)
	if err == nil {
		return state, true, nil
	}
	if err.Error() != "execution not found" {
		return State{}, false, err
	}
	return State{
		Status:      StatusInactive,
		WorkspaceID: string(workspaceID),
	}, false, nil
}
