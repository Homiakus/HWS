package daemon

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
)

func (r *Runtime) RecoverWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, key, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := r.workspaceLifecycle.Recover(ctx, id, key); err != nil {
		return "", err
	}
	return r.WorkspaceStateJSON(string(id))
}

func (r *Runtime) ResumeWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, key, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := r.workspaceLifecycle.Resume(ctx, id, key); err != nil {
		return "", err
	}
	return r.WorkspaceStateJSON(string(id))
}

func (r *Runtime) CloseWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	id, key, err := validateWorkspaceMutation(workspaceID, operationKey)
	if err != nil {
		return "", err
	}
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := r.workspaceLifecycle.Close(ctx, id, key); err != nil {
		return "", err
	}
	return r.WorkspaceStateJSON(string(id))
}

// Suspend is deliberately a state-only operation in lifecycle v1: it leaves
// resources running and marks the workspace inactive. This method exposes that
// exact semantics rather than pretending that resources were paused.
func (r *Runtime) SuspendWorkspaceJSON(workspaceID string) (string, error) {
	id := domain.WorkspaceID(strings.TrimSpace(workspaceID))
	if id == "" {
		return "", errors.New("workspace id is required")
	}
	if r.workspaceLifecycle == nil {
		return "", errors.New("workspace lifecycle is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.workspaceLifecycle.Suspend(ctx, id); err != nil {
		return "", err
	}
	return r.WorkspaceStateJSON(string(id))
}

func validateWorkspaceMutation(workspaceID, operationKey string) (domain.WorkspaceID, string, error) {
	id := domain.WorkspaceID(strings.TrimSpace(workspaceID))
	key := strings.TrimSpace(operationKey)
	if id == "" {
		return "", "", errors.New("workspace id is required")
	}
	if key == "" {
		return "", "", errors.New("workspace operation key is required")
	}
	return id, key, nil
}
