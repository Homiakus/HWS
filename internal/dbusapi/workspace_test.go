package dbusapi

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

type workspaceFakeBackend struct {
	fakeBackend
	calls      []string
	state      string
	states     string
	completion string
}

func (f *workspaceFakeBackend) ActivateWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return f.record("activate", workspaceID, operationKey)
}

func (f *workspaceFakeBackend) RecoverWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return f.record("recover", workspaceID, operationKey)
}

func (f *workspaceFakeBackend) ResumeWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return f.record("resume", workspaceID, operationKey)
}

func (f *workspaceFakeBackend) SuspendWorkspaceJSON(workspaceID string) (string, error) {
	return f.record("suspend", workspaceID, "")
}

func (f *workspaceFakeBackend) CloseWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return f.record("close", workspaceID, operationKey)
}

func (f *workspaceFakeBackend) WorkspaceStateJSON(workspaceID string) (string, error) {
	f.calls = append(f.calls, "state:"+workspaceID)
	return f.state, f.err
}

func (f *workspaceFakeBackend) WorkspaceStatesJSON() (string, error) {
	f.calls = append(f.calls, "states")
	return f.states, f.err
}

func (f *workspaceFakeBackend) CompleteShellActionJSON(payload string) error {
	f.completion = payload
	return f.err
}

func (f *workspaceFakeBackend) record(action, workspaceID, operationKey string) (string, error) {
	call := action + ":" + workspaceID
	if operationKey != "" {
		call += ":" + operationKey
	}
	f.calls = append(f.calls, call)
	return `{"status":"` + action + `"}`, f.err
}

func TestWorkspaceServiceMethods(t *testing.T) {
	backend := &workspaceFakeBackend{
		state:  `{"status":"inactive"}`,
		states: `{"schema":1,"revision":2,"states":[]}`,
	}
	service := NewService(backend)

	value, dbusErr := service.ActivateWorkspace("dev", "op-1")
	assertWorkspaceServiceResult(t, value, dbusErr, `{"status":"activate"}`)
	value, dbusErr = service.RecoverWorkspace("dev", "op-2")
	assertWorkspaceServiceResult(t, value, dbusErr, `{"status":"recover"}`)
	value, dbusErr = service.ResumeWorkspace("dev", "op-3")
	assertWorkspaceServiceResult(t, value, dbusErr, `{"status":"resume"}`)
	value, dbusErr = service.SuspendWorkspace("dev")
	assertWorkspaceServiceResult(t, value, dbusErr, `{"status":"suspend"}`)
	value, dbusErr = service.CloseWorkspace("dev", "op-4")
	assertWorkspaceServiceResult(t, value, dbusErr, `{"status":"close"}`)

	state, dbusErr := service.GetWorkspaceState("dev")
	if dbusErr != nil || state != backend.state {
		t.Fatalf("state=%q err=%v", state, dbusErr)
	}
	states, dbusErr := service.GetWorkspaceStates()
	if dbusErr != nil || states != backend.states {
		t.Fatalf("states=%q err=%v", states, dbusErr)
	}
	if dbusErr := service.CompleteShellAction(`{"schema":1,"id":"x","success":true}`); dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if backend.completion == "" {
		t.Fatal("completion was not forwarded")
	}

	want := []string{
		"activate:dev:op-1",
		"recover:dev:op-2",
		"resume:dev:op-3",
		"suspend:dev",
		"close:dev:op-4",
		"state:dev",
		"states",
	}
	if len(backend.calls) != len(want) {
		t.Fatalf("calls=%#v want=%#v", backend.calls, want)
	}
	for i := range want {
		if backend.calls[i] != want[i] {
			t.Fatalf("call[%d]=%q want=%q", i, backend.calls[i], want[i])
		}
	}
}

func assertWorkspaceServiceResult(t *testing.T, value string, dbusErr *dbus.Error, want string) {
	t.Helper()
	if dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if value != want {
		t.Fatalf("value=%q want=%q", value, want)
	}
}
