package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Homiakus/HWS/internal/domain"
)

func TestWorkspaceFileKeepsVersionedDefinitionsAndActiveRevision(t *testing.T) {
	file := NewFile()
	const source = `{
  "schema":1,
  "active":{"dev":"v2"},
  "workspaces":[
    {"id":"dev","revision":"v1","resources":[]},
    {"id":"dev","revision":"v2","resources":[
      {"id":"editor","kind":"desktop_app","required":true,"ownership":"managed","desktopAppId":"dev.zed.Zed.desktop"}
    ]}
  ]
}`
	if _, err := file.Apply([]byte(source)); err != nil {
		t.Fatal(err)
	}
	current, err := file.Current("dev")
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != "v2" || len(current.Resources) != 1 {
		t.Fatalf("unexpected current definition: %#v", current)
	}
	old, err := file.Resolve("dev", "v1")
	if err != nil {
		t.Fatal(err)
	}
	if old.Revision != "v1" {
		t.Fatalf("old revision lost: %#v", old)
	}
}

func TestWorkspaceFileRejectsDanglingActiveRevision(t *testing.T) {
	file := NewFile()
	_, err := file.Apply([]byte(`{"schema":1,"active":{"dev":"v2"},"workspaces":[{"id":"dev","revision":"v1","resources":[]}]}`))
	if err == nil {
		t.Fatal("dangling active revision accepted")
	}
}

func TestInvalidReloadKeepsLastKnownGoodWorkspaceCatalog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "workspaces.json")
	file := NewFile()
	if err := file.Configure(path); err != nil {
		t.Fatal(err)
	}
	valid := `{"schema":1,"active":{"dev":"v1"},"workspaces":[{"id":"dev","revision":"v1","resources":[]}]}`
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Reload(); err != nil {
		t.Fatal(err)
	}
	before := file.Revision()
	if err := os.WriteFile(path, []byte(`{"schema":1,"active":{"dev":"missing"},"workspaces":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := file.Reload()
	if err == nil || changed {
		t.Fatalf("invalid reload accepted: changed=%v err=%v", changed, err)
	}
	if file.Valid() {
		t.Fatal("invalid current file reported valid")
	}
	if file.Revision() != before {
		t.Fatalf("revision changed after rejected reload: before=%d after=%d", before, file.Revision())
	}
	if _, err := file.Current(domain.WorkspaceID("dev")); err != nil {
		t.Fatalf("last-known-good definition unavailable: %v", err)
	}
}

func TestWorkspaceFileStrictlyRejectsUnknownFields(t *testing.T) {
	file := NewFile()
	_, err := file.Apply([]byte(`{"schema":1,"active":{},"workspaces":[],"exec":"rm -rf /"}`))
	if err == nil {
		t.Fatal("unknown field accepted")
	}
}
