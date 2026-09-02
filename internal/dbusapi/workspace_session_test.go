package dbusapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/daemon"
	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/Homiakus/HWS/internal/shellaction"
	"github.com/godbus/dbus/v5"
)

func TestSessionRoundTripWorkspaceActivationAndClose(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("requires a session bus; CI runs this test under dbus-run-session")
	}

	dir := t.TempDir()
	workspacePath := filepath.Join(dir, "workspaces.json")
	const workspaceConfig = `{
  "schema":1,
  "active":{"dev":"v1"},
  "workspaces":[{
    "id":"dev",
    "revision":"v1",
    "resources":[{
      "id":"editor",
      "kind":"desktop_app",
      "required":true,
      "ownership":"managed",
      "desktopAppId":"dev.zed.Zed.desktop"
    }]
  }]
}`
	if err := os.WriteFile(workspacePath, []byte(workspaceConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	runtime := daemon.NewRuntime(nil)
	if err := runtime.ConfigureWorkspaces(workspacePath); err != nil {
		t.Fatal(err)
	}
	if err := runtime.OpenWorkspaceLifecycle(filepath.Join(dir, "axiom")); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()

	server, err := OpenSession(runtime)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	runtime.SetShellActionEmitter(server.EmitShellActionRequested)
	runtime.SetWorkspaceNotifier(server.EmitWorkspaceChanged)

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	shellMatch := []dbus.MatchOption{
		dbus.WithMatchObjectPath(dbus.ObjectPath(ipc.ObjectPath)),
		dbus.WithMatchInterface(ipc.InterfaceName),
		dbus.WithMatchMember("ShellActionRequested"),
	}
	workspaceMatch := []dbus.MatchOption{
		dbus.WithMatchObjectPath(dbus.ObjectPath(ipc.ObjectPath)),
		dbus.WithMatchInterface(ipc.InterfaceName),
		dbus.WithMatchMember("WorkspaceChanged"),
	}
	if err := conn.AddMatchSignal(shellMatch...); err != nil {
		t.Fatal(err)
	}
	defer conn.RemoveMatchSignal(shellMatch...)
	if err := conn.AddMatchSignal(workspaceMatch...); err != nil {
		t.Fatal(err)
	}
	defer conn.RemoveMatchSignal(workspaceMatch...)

	signals := make(chan *dbus.Signal, 12)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	obj := conn.Object(ipc.BusName, dbus.ObjectPath(ipc.ObjectPath))

	initialRevision := assertWorkspaceStates(t, obj, "inactive")

	activateResult := asyncWorkspaceCall(obj, "ActivateWorkspace", "dev", "integration:dev:v1:activate")
	ensure := waitShellAction(t, signals)
	if ensure.Kind != shellaction.KindEnsureDesktopApp || ensure.DesktopAppID != "dev.zed.Zed.desktop" {
		t.Fatalf("unexpected ensure request: %#v", ensure)
	}

	snapshot := `{"schema":1,"revision":1,"capturedAt":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","apps":[{"appId":"dev.zed.Zed.desktop","name":"Zed","desktopAppId":"dev.zed.Zed.desktop","windows":[{"id":"window:7","title":"HWS"}]}]}`
	if err := obj.Call(ipc.InterfaceName+".SubmitShellSnapshot", 0, snapshot).Err; err != nil {
		t.Fatalf("SubmitShellSnapshot active: %v", err)
	}
	completeShellAction(t, obj, ensure, true, true)
	assertWorkspaceResult(t, activateResult, "active")
	activeRevision := waitWorkspaceChanged(t, signals, "dev")
	if activeRevision <= initialRevision {
		t.Fatalf("active revision=%d initial=%d", activeRevision, initialRevision)
	}
	if got := assertWorkspaceStates(t, obj, "active"); got != activeRevision {
		t.Fatalf("batch revision=%d signal revision=%d", got, activeRevision)
	}

	closeResult := asyncWorkspaceCall(obj, "CloseWorkspace", "dev", "integration:dev:v1:close")
	closeRequest := waitShellAction(t, signals)
	if closeRequest.Kind != shellaction.KindCloseWindow || closeRequest.WindowID != "window:7" {
		t.Fatalf("unexpected close request: %#v", closeRequest)
	}

	emptySnapshot := `{"schema":1,"revision":2,"capturedAt":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","apps":[]}`
	if err := obj.Call(ipc.InterfaceName+".SubmitShellSnapshot", 0, emptySnapshot).Err; err != nil {
		t.Fatalf("SubmitShellSnapshot closed: %v", err)
	}
	completeShellAction(t, obj, closeRequest, true, true)
	assertWorkspaceResult(t, closeResult, "inactive")
	closedRevision := waitWorkspaceChanged(t, signals, "dev")
	if closedRevision <= activeRevision {
		t.Fatalf("closed revision=%d active=%d", closedRevision, activeRevision)
	}
	if got := assertWorkspaceStates(t, obj, "inactive"); got != closedRevision {
		t.Fatalf("batch revision=%d signal revision=%d", got, closedRevision)
	}

	var state string
	if err := obj.Call(ipc.InterfaceName+".GetWorkspaceState", 0, "dev").Store(&state); err != nil {
		t.Fatalf("GetWorkspaceState: %v", err)
	}
	assertWorkspaceJSONStatus(t, state, "inactive")
}

type workspaceCallResult struct {
	state string
	err   error
}

func asyncWorkspaceCall(obj dbus.BusObject, method string, args ...any) <-chan workspaceCallResult {
	result := make(chan workspaceCallResult, 1)
	go func() {
		var state string
		err := obj.Call(ipc.InterfaceName+"."+method, 0, args...).Store(&state)
		result <- workspaceCallResult{state: state, err: err}
	}()
	return result
}

func waitShellAction(t *testing.T, signals <-chan *dbus.Signal) shellaction.Request {
	t.Helper()
	for {
		select {
		case signal := <-signals:
			if signal == nil || signal.Name != ipc.InterfaceName+".ShellActionRequested" {
				continue
			}
			if len(signal.Body) != 1 {
				t.Fatalf("unexpected signal body: %#v", signal)
			}
			payload, ok := signal.Body[0].(string)
			if !ok {
				t.Fatalf("unexpected shell action body: %#v", signal.Body)
			}
			var request shellaction.Request
			if err := json.Unmarshal([]byte(payload), &request); err != nil {
				t.Fatal(err)
			}
			return request
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for ShellActionRequested")
			return shellaction.Request{}
		}
	}
}

func waitWorkspaceChanged(t *testing.T, signals <-chan *dbus.Signal, wantID string) uint64 {
	t.Helper()
	for {
		select {
		case signal := <-signals:
			if signal == nil || signal.Name != ipc.InterfaceName+".WorkspaceChanged" {
				continue
			}
			if len(signal.Body) != 2 {
				t.Fatalf("unexpected WorkspaceChanged body: %#v", signal.Body)
			}
			workspaceID, ok := signal.Body[0].(string)
			if !ok || workspaceID != wantID {
				t.Fatalf("workspace changed id=%#v want=%q", signal.Body[0], wantID)
			}
			revision, ok := signal.Body[1].(uint64)
			if !ok {
				t.Fatalf("workspace changed revision=%#v", signal.Body[1])
			}
			return revision
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for WorkspaceChanged")
			return 0
		}
	}
}

func assertWorkspaceStates(t *testing.T, obj dbus.BusObject, wantStatus string) uint64 {
	t.Helper()
	var raw string
	if err := obj.Call(ipc.InterfaceName+".GetWorkspaceStates", 0).Store(&raw); err != nil {
		t.Fatalf("GetWorkspaceStates: %v", err)
	}
	var snapshot daemon.WorkspaceStatesSnapshot
	if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Schema != 1 || len(snapshot.States) != 1 {
		t.Fatalf("unexpected workspace states snapshot: %s", raw)
	}
	if snapshot.States[0].WorkspaceID != "dev" || snapshot.States[0].Status != wantStatus {
		t.Fatalf("unexpected workspace state snapshot: %s", raw)
	}
	return snapshot.Revision
}

func completeShellAction(t *testing.T, obj dbus.BusObject, request shellaction.Request, success, changed bool) {
	t.Helper()
	completion, err := json.Marshal(shellaction.Result{
		Schema:  shellaction.SchemaVersion,
		ID:      request.ID,
		Success: success,
		Changed: changed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := obj.Call(ipc.InterfaceName+".CompleteShellAction", 0, string(completion)).Err; err != nil {
		t.Fatalf("CompleteShellAction: %v", err)
	}
}

func assertWorkspaceResult(t *testing.T, result <-chan workspaceCallResult, wantStatus string) {
	t.Helper()
	select {
	case call := <-result:
		if call.err != nil {
			t.Fatal(call.err)
		}
		assertWorkspaceJSONStatus(t, call.state, wantStatus)
	case <-time.After(3 * time.Second):
		t.Fatalf("workspace call did not complete with status %s", wantStatus)
	}
}

func assertWorkspaceJSONStatus(t *testing.T, raw, wantStatus string) {
	t.Helper()
	var state struct {
		Status      string `json:"status"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatal(err)
	}
	if state.Status != wantStatus || state.WorkspaceID != "dev" {
		t.Fatalf("unexpected state: %s", raw)
	}
}
