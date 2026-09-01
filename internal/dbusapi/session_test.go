package dbusapi

import (
	"os"
	"testing"

	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/godbus/dbus/v5"
)

func TestSessionRoundTrip(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("requires a session bus; CI runs this test under dbus-run-session")
	}
	server, err := OpenSession(&fakeBackend{panel: `{"revision":1,"cards":[]}`, tree: `{"rootId":"root"}`})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	obj := conn.Object(ipc.BusName, dbus.ObjectPath(ipc.ObjectPath))

	var protocol uint32
	var instance, epoch, capabilities string
	if err := obj.Call(ipc.InterfaceName+".Hello", 0, ipc.ProtocolVersion, "integration-test").Store(&protocol, &instance, &epoch, &capabilities); err != nil {
		t.Fatalf("Hello: %v", err)
	}
	if protocol != ipc.ProtocolVersion || instance == "" || epoch == "" || capabilities == "" {
		t.Fatalf("invalid hello: protocol=%d instance=%q epoch=%q caps=%q", protocol, instance, epoch, capabilities)
	}

	var panelJSON string
	if err := obj.Call(ipc.InterfaceName+".GetPanelSnapshot", 0).Store(&panelJSON); err != nil {
		t.Fatalf("GetPanelSnapshot: %v", err)
	}
	if panelJSON != `{"revision":1,"cards":[]}` {
		t.Fatalf("panel=%q", panelJSON)
	}

	var treeJSON string
	if err := obj.Call(ipc.InterfaceName+".GetTree", 0).Store(&treeJSON); err != nil {
		t.Fatalf("GetTree: %v", err)
	}
	if treeJSON != `{"rootId":"root"}` {
		t.Fatalf("tree=%q", treeJSON)
	}
}

func TestGoClientRoundTrip(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("requires a session bus; CI runs this test under dbus-run-session")
	}
	backend := &fakeBackend{
		panel:  `{"revision":9,"cards":[]}`,
		spec:   `{"revision":2,"valid":true}`,
		health: `{"status":"ok","applications":1}`,
		tree:   `{"schema":1,"revision":3,"rootId":"root","nodes":[]}`,
		path:   `[{"id":"root","title":"Home"}]`,
		app:    `{"appId":"code.desktop"}`,
	}
	server, err := OpenSession(backend)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	client, err := ConnectSession()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.ServerInstance() == "" || client.RevisionEpoch() == "" {
		t.Fatal("client handshake did not preserve server cache identity")
	}
	if client.Capabilities()["health"] != "supported" || client.Capabilities()["tree.read"] != "supported" {
		t.Fatalf("required capabilities missing: %#v", client.Capabilities())
	}
	if got, err := client.HealthJSON(); err != nil || got != backend.health {
		t.Fatalf("health=%q err=%v", got, err)
	}
	if got, err := client.PanelJSON(); err != nil || got != backend.panel {
		t.Fatalf("panel=%q err=%v", got, err)
	}
	if got, err := client.SpecJSON(); err != nil || got != backend.spec {
		t.Fatalf("spec=%q err=%v", got, err)
	}
	if got, err := client.TreeJSON(); err != nil || got != backend.tree {
		t.Fatalf("tree=%q err=%v", got, err)
	}
	if got, err := client.PathJSON("root"); err != nil || got != backend.path {
		t.Fatalf("path=%q err=%v", got, err)
	}
	if got, err := client.ApplicationJSON("code.desktop"); err != nil || got != backend.app {
		t.Fatalf("app=%q err=%v", got, err)
	}
}
