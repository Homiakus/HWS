package dbusapi

import "testing"

type workspaceFakeBackend struct {
	fakeBackend
	activation string
	state      string
	completion string
}

func (f *workspaceFakeBackend) ActivateWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	f.activation = workspaceID + ":" + operationKey
	return `{"status":"active"}`, f.err
}

func (f *workspaceFakeBackend) WorkspaceStateJSON(string) (string, error) {
	return f.state, f.err
}

func (f *workspaceFakeBackend) CompleteShellActionJSON(payload string) error {
	f.completion = payload
	return f.err
}

func TestWorkspaceServiceMethods(t *testing.T) {
	backend := &workspaceFakeBackend{state: `{"status":"inactive"}`}
	service := NewService(backend)
	state, dbusErr := service.ActivateWorkspace("dev", "op-1")
	if dbusErr != nil || state != `{"status":"active"}` || backend.activation != "dev:op-1" {
		t.Fatalf("activate state=%q backend=%q err=%v", state, backend.activation, dbusErr)
	}
	state, dbusErr = service.GetWorkspaceState("dev")
	if dbusErr != nil || state != backend.state {
		t.Fatalf("state=%q err=%v", state, dbusErr)
	}
	if dbusErr := service.CompleteShellAction(`{"schema":1,"id":"x","success":true}`); dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if backend.completion == "" {
		t.Fatal("completion was not forwarded")
	}
}
