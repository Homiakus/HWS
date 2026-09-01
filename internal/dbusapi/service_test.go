package dbusapi

import (
	"errors"
	"testing"

	"github.com/Homiakus/HWS/internal/ipc"
)

type fakeBackend struct {
	panel         string
	spec          string
	health        string
	tree          string
	path          string
	app           string
	shellSnapshot string
	err           error
}

func (f *fakeBackend) PanelJSON() (string, error)             { return f.panel, f.err }
func (f *fakeBackend) SpecJSON() (string, error)              { return f.spec, f.err }
func (f *fakeBackend) HealthJSON() (string, error)            { return f.health, f.err }
func (f *fakeBackend) TreeJSON() (string, error)              { return f.tree, f.err }
func (f *fakeBackend) PathJSON(string) (string, error)        { return f.path, f.err }
func (f *fakeBackend) ApplicationJSON(string) (string, error) { return f.app, f.err }
func (f *fakeBackend) ReplaceShellSnapshotJSON(value string) error {
	f.shellSnapshot = value
	return f.err
}
func (f *fakeBackend) ReloadPanel() error                { return f.err }
func (f *fakeBackend) ActivateView(string, string) error { return f.err }
func (f *fakeBackend) CloseView(string, string) error    { return f.err }

func TestHelloNegotiatesProtocol(t *testing.T) {
	s := NewService(&fakeBackend{})
	protocol, instance, epoch, capabilities, dbusErr := s.Hello(ipc.ProtocolVersion, "test-client")
	if dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if protocol != ipc.ProtocolVersion || instance == "" || epoch == "" || capabilities == "" {
		t.Fatalf("invalid hello response: %d %q %q %q", protocol, instance, epoch, capabilities)
	}
	if _, _, _, _, err := s.Hello(ipc.ProtocolVersion+1, "test-client"); err == nil {
		t.Fatal("protocol mismatch accepted")
	}
}

func TestServicePreservesReloadDiagnostics(t *testing.T) {
	s := NewService(&fakeBackend{err: errors.New("bad panel")})
	ok, diagnostic, dbusErr := s.ReloadPanel()
	if dbusErr != nil {
		t.Fatal(dbusErr)
	}
	if ok || diagnostic != "bad panel" {
		t.Fatalf("unexpected reload result: ok=%v diagnostic=%q", ok, diagnostic)
	}
}

func TestServiceAcceptsShellSnapshot(t *testing.T) {
	backend := &fakeBackend{}
	s := NewService(backend)
	const payload = `{"schema":1}`
	if err := s.SubmitShellSnapshot(payload); err != nil {
		t.Fatal(err)
	}
	if backend.shellSnapshot != payload {
		t.Fatalf("snapshot=%q", backend.shellSnapshot)
	}
}

func TestServiceExposesHierarchy(t *testing.T) {
	backend := &fakeBackend{tree: `{"rootId":"root"}`, path: `[{"id":"root"}]`}
	s := NewService(backend)
	if got, err := s.GetTree(); err != nil || got != backend.tree {
		t.Fatalf("tree=%q err=%v", got, err)
	}
	if got, err := s.GetPath("root"); err != nil || got != backend.path {
		t.Fatalf("path=%q err=%v", got, err)
	}
}
