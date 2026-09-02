package dbusapi

import "testing"

type workspaceFakeBackend struct {
	fakeBackend
	calls      []string
	state      string
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
	backend := &workspaceFakeBackend{state: `{"status":"inactive"}`}
	service := NewService(backend)

	assertServiceWorkspaceCall(t, service.ActivateWorkspace("dev", "op-1"), `{"status":"activate"}`)
	assertServiceWorkspaceCall(t, service.RecoverWorkspace("dev", "op-2"), `{"status":"recover"}`)
	assertServiceWorkspaceCall(t, service.ResumeWorkspace("dev", "op-3"), `{"status":"resume"}`)
	assertServiceWorkspaceCall(t, service.SuspendWorkspace("dev"), `{"status":"suspend"}`)
	assertServiceWorkspaceCall(t, service.CloseWorkspace("dev", "op-4"), `{"status":"close"}`)

	state, dbusErr := service.GetWorkspaceState("dev")
	if dbusErr != nil || state != backend.state {
		t.Fatalf("state=%q err=%v", state, dbusErr)
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

func assertServiceWorkspaceCall(t *testing.T, value string, dbusErr interface{ Error() string }, want string) {
	t.Helper()
	if dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if value != want {
		t.Fatalf("value=%q want=%q", value, want)
	}
}
