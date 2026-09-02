package dbusapi

import (
	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/godbus/dbus/v5"
)

type workspaceBackend interface {
	ActivateWorkspaceJSON(string, string) (string, error)
	RecoverWorkspaceJSON(string, string) (string, error)
	ResumeWorkspaceJSON(string, string) (string, error)
	SuspendWorkspaceJSON(string) (string, error)
	CloseWorkspaceJSON(string, string) (string, error)
	WorkspaceStateJSON(string) (string, error)
	CompleteShellActionJSON(string) error
}

func (s *Service) ActivateWorkspace(workspaceID, operationKey string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.ActivateWorkspaceJSON(workspaceID, operationKey)
	return value, asDBusError(callErr)
}

func (s *Service) RecoverWorkspace(workspaceID, operationKey string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.RecoverWorkspaceJSON(workspaceID, operationKey)
	return value, asDBusError(callErr)
}

func (s *Service) ResumeWorkspace(workspaceID, operationKey string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.ResumeWorkspaceJSON(workspaceID, operationKey)
	return value, asDBusError(callErr)
}

func (s *Service) SuspendWorkspace(workspaceID string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.SuspendWorkspaceJSON(workspaceID)
	return value, asDBusError(callErr)
}

func (s *Service) CloseWorkspace(workspaceID, operationKey string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.CloseWorkspaceJSON(workspaceID, operationKey)
	return value, asDBusError(callErr)
}

func (s *Service) GetWorkspaceState(workspaceID string) (string, *dbus.Error) {
	backend, err := s.workspaceBackend()
	if err != nil {
		return "", err
	}
	value, callErr := backend.WorkspaceStateJSON(workspaceID)
	return value, asDBusError(callErr)
}

func (s *Service) CompleteShellAction(payload string) *dbus.Error {
	backend, err := s.workspaceBackend()
	if err != nil {
		return err
	}
	return asDBusError(backend.CompleteShellActionJSON(payload))
}

func (s *Service) workspaceBackend() (workspaceBackend, *dbus.Error) {
	backend, ok := s.backend.(workspaceBackend)
	if !ok {
		return nil, dbus.NewError(ipc.InterfaceName+".Unsupported", []any{"workspace lifecycle is unavailable"})
	}
	return backend, nil
}

func (s *Server) EmitShellActionRequested(payload string) {
	if payload == "" {
		return
	}
	s.emit("ShellActionRequested", payload)
}
