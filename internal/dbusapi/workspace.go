package dbusapi

import (
	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/godbus/dbus/v5"
)

type workspaceBackend interface {
	ActivateWorkspaceJSON(string, string) (string, error)
	WorkspaceStateJSON(string) (string, error)
	CompleteShellActionJSON(string) error
}

func (s *Service) ActivateWorkspace(workspaceID, operationKey string) (string, *dbus.Error) {
	backend, ok := s.backend.(workspaceBackend)
	if !ok {
		return "", dbus.NewError(ipc.InterfaceName+".Unsupported", []any{"workspace activation is unavailable"})
	}
	value, err := backend.ActivateWorkspaceJSON(workspaceID, operationKey)
	return value, asDBusError(err)
}

func (s *Service) GetWorkspaceState(workspaceID string) (string, *dbus.Error) {
	backend, ok := s.backend.(workspaceBackend)
	if !ok {
		return "", dbus.NewError(ipc.InterfaceName+".Unsupported", []any{"workspace state is unavailable"})
	}
	value, err := backend.WorkspaceStateJSON(workspaceID)
	return value, asDBusError(err)
}

func (s *Service) CompleteShellAction(payload string) *dbus.Error {
	backend, ok := s.backend.(workspaceBackend)
	if !ok {
		return dbus.NewError(ipc.InterfaceName+".Unsupported", []any{"shell action completion is unavailable"})
	}
	return asDBusError(backend.CompleteShellActionJSON(payload))
}

func (s *Server) EmitShellActionRequested(payload string) {
	if payload == "" {
		return
	}
	s.emit("ShellActionRequested", payload)
}
