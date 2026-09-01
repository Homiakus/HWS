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
	server, err := OpenSession(&fakeBackend{panel: `{"revision":1,"cards":[]}`})
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
}
