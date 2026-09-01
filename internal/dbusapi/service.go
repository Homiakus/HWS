package dbusapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
)

type Backend interface {
	PanelJSON() (string, error)
	SpecJSON() (string, error)
	HealthJSON() (string, error)
	TreeJSON() (string, error)
	PathJSON(string) (string, error)
	ApplicationJSON(string) (string, error)
	ReplaceShellSnapshotJSON(string) error
	ReloadPanel() error
	ActivateView(string, string) error
	CloseView(string, string) error
}

type Service struct {
	backend        Backend
	serverInstance string
	revisionEpoch  string
}

func NewService(backend Backend) *Service {
	return &Service{backend: backend, serverInstance: randomID(), revisionEpoch: randomID()}
}

func randomID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "hws-instance"
	}
	return hex.EncodeToString(data[:])
}

func (s *Service) Hello(clientProtocol uint32, clientInstance string) (uint32, string, string, string, *dbus.Error) {
	if clientProtocol != ipc.ProtocolVersion {
		return 0, "", "", "", dbus.NewError(ipc.InterfaceName+".ProtocolIncompatible", []any{fmt.Sprintf("client protocol %d is not supported", clientProtocol)})
	}
	if clientInstance == "" {
		return 0, "", "", "", dbus.NewError(ipc.InterfaceName+".InvalidClient", []any{"client instance is required"})
	}
	capabilities, _ := json.Marshal(map[string]string{
		"health":              "supported",
		"panel.snapshot":      "supported",
		"panel.spec":          "supported",
		"panel.reload":        "supported",
		"tree.read":           "supported",
		"shell.snapshot.push": "supported",
		"surface.application": "supported",
		"view.activate":       "supported",
		"view.close":          "supported",
	})
	return ipc.ProtocolVersion, s.serverInstance, s.revisionEpoch, string(capabilities), nil
}

func (s *Service) GetPanelSnapshot() (string, *dbus.Error) {
	value, err := s.backend.PanelJSON()
	return value, asDBusError(err)
}

func (s *Service) GetPanelSpec() (string, *dbus.Error) {
	value, err := s.backend.SpecJSON()
	return value, asDBusError(err)
}

func (s *Service) GetHealth() (string, *dbus.Error) {
	value, err := s.backend.HealthJSON()
	return value, asDBusError(err)
}

func (s *Service) GetTree() (string, *dbus.Error) {
	value, err := s.backend.TreeJSON()
	return value, asDBusError(err)
}

func (s *Service) GetPath(nodeID string) (string, *dbus.Error) {
	value, err := s.backend.PathJSON(nodeID)
	return value, asDBusError(err)
}

func (s *Service) GetApplicationSurface(appID string) (string, *dbus.Error) {
	value, err := s.backend.ApplicationJSON(appID)
	return value, asDBusError(err)
}

func (s *Service) SubmitShellSnapshot(payload string) *dbus.Error {
	return asDBusError(s.backend.ReplaceShellSnapshotJSON(payload))
}

func (s *Service) ReloadPanel() (bool, string, *dbus.Error) {
	if err := s.backend.ReloadPanel(); err != nil {
		return false, err.Error(), nil
	}
	return true, "", nil
}

func (s *Service) ActivateView(appID, viewID string) *dbus.Error {
	return asDBusError(s.backend.ActivateView(appID, viewID))
}

func (s *Service) CloseView(appID, viewID string) *dbus.Error {
	return asDBusError(s.backend.CloseView(appID, viewID))
}

func asDBusError(err error) *dbus.Error {
	if err == nil {
		return nil
	}
	return dbus.NewError(ipc.InterfaceName+".Error", []any{err.Error()})
}

type Server struct {
	mu      sync.Mutex
	conn    *dbus.Conn
	service *Service
	closed  bool
}

func OpenSession(backend Backend) (*Server, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}
	service := NewService(backend)
	path := dbus.ObjectPath(ipc.ObjectPath)
	if err := conn.Export(service, path, ipc.InterfaceName); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("export HWS interface: %w", err)
	}
	node := introspectionNode()
	if err := conn.Export(introspect.NewIntrospectable(node), path, "org.freedesktop.DBus.Introspectable"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("export introspection: %w", err)
	}
	reply, err := conn.RequestName(ipc.BusName, dbus.NameFlagDoNotQueue)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("request D-Bus name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return nil, fmt.Errorf("D-Bus name %s is already owned", ipc.BusName)
	}
	return &Server{conn: conn, service: service}, nil
}

func (s *Server) EmitPanelChanged(revision uint64) {
	s.emit("PanelChanged", revision)
}

func (s *Server) EmitPanelConfigChanged(revision uint64) {
	s.emit("PanelConfigChanged", revision)
}

func (s *Server) EmitTreeChanged(revision uint64) {
	s.emit("TreeChanged", revision)
}

func (s *Server) emit(signal string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.conn == nil {
		return
	}
	_ = s.conn.Emit(dbus.ObjectPath(ipc.ObjectPath), ipc.InterfaceName+"."+signal, args...)
}

func (s *Server) Run(ctx context.Context) {
	<-ctx.Done()
	_ = s.Close()
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.conn == nil {
		return nil
	}
	_, _ = s.conn.ReleaseName(ipc.BusName)
	return s.conn.Close()
}

func introspectionNode() *introspect.Node {
	return &introspect.Node{
		Name: ipc.ObjectPath,
		Interfaces: []introspect.Interface{
			{
				Name: ipc.InterfaceName,
				Methods: []introspect.Method{
					{Name: "Hello", Args: []introspect.Arg{{Name: "clientProtocol", Type: "u", Direction: "in"}, {Name: "clientInstance", Type: "s", Direction: "in"}, {Name: "serverProtocol", Type: "u", Direction: "out"}, {Name: "serverInstance", Type: "s", Direction: "out"}, {Name: "revisionEpoch", Type: "s", Direction: "out"}, {Name: "capabilities", Type: "s", Direction: "out"}}},
					{Name: "GetPanelSnapshot", Args: []introspect.Arg{{Name: "json", Type: "s", Direction: "out"}}},
					{Name: "GetPanelSpec", Args: []introspect.Arg{{Name: "json", Type: "s", Direction: "out"}}},
					{Name: "GetHealth", Args: []introspect.Arg{{Name: "json", Type: "s", Direction: "out"}}},
					{Name: "GetTree", Args: []introspect.Arg{{Name: "json", Type: "s", Direction: "out"}}},
					{Name: "GetPath", Args: []introspect.Arg{{Name: "nodeId", Type: "s", Direction: "in"}, {Name: "json", Type: "s", Direction: "out"}}},
					{Name: "GetApplicationSurface", Args: []introspect.Arg{{Name: "appId", Type: "s", Direction: "in"}, {Name: "json", Type: "s", Direction: "out"}}},
					{Name: "SubmitShellSnapshot", Args: []introspect.Arg{{Name: "json", Type: "s", Direction: "in"}}},
					{Name: "ReloadPanel", Args: []introspect.Arg{{Name: "ok", Type: "b", Direction: "out"}, {Name: "diagnostic", Type: "s", Direction: "out"}}},
					{Name: "ActivateView", Args: []introspect.Arg{{Name: "appId", Type: "s", Direction: "in"}, {Name: "viewId", Type: "s", Direction: "in"}}},
					{Name: "CloseView", Args: []introspect.Arg{{Name: "appId", Type: "s", Direction: "in"}, {Name: "viewId", Type: "s", Direction: "in"}}},
				},
				Signals: []introspect.Signal{
					{Name: "PanelChanged", Args: []introspect.Arg{{Name: "revision", Type: "t"}}},
					{Name: "PanelConfigChanged", Args: []introspect.Arg{{Name: "revision", Type: "t"}}},
					{Name: "TreeChanged", Args: []introspect.Arg{{Name: "revision", Type: "t"}}},
				},
			},
			introspect.IntrospectData,
		},
	}
}
