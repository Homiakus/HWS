package daemon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/shellaction"
)

func TestRuntimeActivatesWorkspaceThroughAxiomAndShellActionBroker(t *testing.T) {
	runtime := NewRuntime(NewHub(providers.NewRegistry()))
	workspacePath := filepath.Join(t.TempDir(), "workspaces.json")
	if err := runtime.ConfigureWorkspaces(workspacePath); err != nil {
		t.Fatal(err)
	}
	const workspaces = `{
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
	if err := os.WriteFile(workspacePath, []byte(workspaces), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.workspaces.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.OpenWorkspaceLifecycle(filepath.Join(t.TempDir(), "axiom")); err != nil {
		t.Fatal(err)
	}
	defer runtime.Shutdown()

	runtime.SetShellActionEmitter(func(payload string) {
		var request shellaction.Request
		if err := json.Unmarshal([]byte(payload), &request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if request.Kind != shellaction.KindEnsureDesktopApp || request.DesktopAppID != "dev.zed.Zed.desktop" {
			t.Errorf("unexpected shell action: %#v", request)
			return
		}
		snapshot := `{"schema":1,"revision":1,"capturedAt":"` + time.Now().UTC().Format(time.RFC3339Nano) + `","apps":[{"appId":"dev.zed.Zed.desktop","name":"Zed","desktopAppId":"dev.zed.Zed.desktop","windows":[{"id":"window:7","title":"HWS"}]}]}`
		if err := runtime.ReplaceShellSnapshotJSON(snapshot); err != nil {
			t.Errorf("replace shell snapshot: %v", err)
			return
		}
		result, err := json.Marshal(shellaction.Result{
			Schema:  shellaction.SchemaVersion,
			ID:      request.ID,
			Success: true,
			Changed: true,
		})
		if err != nil {
			t.Errorf("encode shell result: %v", err)
			return
		}
		if err := runtime.CompleteShellActionJSON(string(result)); err != nil {
			t.Errorf("complete shell action: %v", err)
		}
	})

	stateJSON, err := runtime.ActivateWorkspaceJSON("dev", "test:activate:dev:v1")
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Status      string `json:"status"`
		WorkspaceID string `json:"workspaceId"`
	}
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		t.Fatal(err)
	}
	if state.Status != "active" || state.WorkspaceID != "dev" {
		t.Fatalf("unexpected workspace state: %s", stateJSON)
	}
}
