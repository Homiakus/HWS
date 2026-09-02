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

func TestSessionRoundTripWorkspaceActivation(t *testing.T) {
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

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	match := []dbus.MatchOption{
		dbus.WithMatchObjectPath(dbus.ObjectPath(ipc.ObjectPath)),
		dbus.WithMatchInterface(ipc.InterfaceName),
		dbus.WithMatchMember("ShellActionRequested"),
	}
	if err := conn.AddMatchSignal(match...); err != nil {
		t.Fatal(err)
	}
	defer conn.RemoveMatchSignal(match...)

	signals := make(chan *dbus.Signal, 4)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	obj := conn.Object(ipc.BusName, dbus.ObjectPath(ipc.ObjectPath))

	type activationResult struct {
		state string
		err   error
	}
	activated := make(chan activationResult, 1)
	go func() {
		var state string
		err := obj.Call(ipc.InterfaceName+".ActivateWorkspace", 0, "dev", "integration:dev:v1").Store(&state)
		activated <- activationResult{state: state, err: err}
	}()

	var request shellaction.Request
	select {
	case signal := <-signals:
		if signal == nil || signal.Name != ipc.InterfaceName+".ShellActionRequested" || len(signal.Body) != 1 {
			t.Fatalf("unexpected signal: %#v", signal)
		}
		payload, ok := signal.Body[0].(string)
		if !ok {
			t.Fatalf("unexpected shell action body: %#v", signal.Body)
		}
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for ShellActionRequested")
	}
	if request.Kind != shellaction.KindEnsureDesktopApp || request.DesktopAppID != "dev.zed.Zed.desktop" {
		t.Fatalf("unexpected shell action request: %#v", request)
	}

	snapshot := `{"schema":1,"revision":1,"capturedAt":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","apps":[{"appId":"dev.zed.Zed.desktop","name":"Zed","desktopAppId":"dev.zed.Zed.desktop","windows":[{"id":"window:7","title":"HWS"}]}]}`
	if err := obj.Call(ipc.InterfaceName+".SubmitShellSnapshot", 0, snapshot).Err; err != nil {
		t.Fatalf("SubmitShellSnapshot: %v", err)
	}
	completion, err := json.Marshal(shellaction.Result{
		Schema:  shellaction.SchemaVersion,
		ID:      request.ID,
		Success: true,
		Changed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := obj.Call(ipc.InterfaceName+".CompleteShellAction", 0, string(completion)).Err; err != nil {
		t.Fatalf("CompleteShellAction: %v", err)
	}

	select {
	case result := <-activated:
		if result.err != nil {
			t.Fatal(result.err)
		}
		var state struct {
			Status      string `json:"status"`
			WorkspaceID string `json:"workspaceId"`
		}
		if err := json.Unmarshal([]byte(result.state), &state); err != nil {
			t.Fatal(err)
		}
		if state.Status != "active" || state.WorkspaceID != "dev" {
			t.Fatalf("unexpected state: %s", result.state)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("workspace activation did not complete")
	}
}
