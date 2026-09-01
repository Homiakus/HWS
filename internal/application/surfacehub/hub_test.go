package surfacehub

import (
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/surface"
)

type sender struct {
	provider string
	command  map[string]any
}

func (s *sender) Send(providerID string, command any) error {
	s.provider = providerID
	s.command = command.(map[string]any)
	return nil
}

func TestActivateBrowserViewRoutesProviderCommand(t *testing.T) {
	now := time.Now()
	r := providers.NewRegistry()
	s := &sender{}
	h := New(r, s)
	if err := h.Ingest(providers.Snapshot{ProviderID: "browser:firefox", AppID: "firefox.desktop", AllowOrphan: true, ApplicationName: "Firefox", ObservedAt: now, TTL: time.Minute, Revision: 1, Windows: []providers.WindowPatch{{Views: []surface.View{{ID: "tab:42", Kind: surface.ViewTab}}}}}); err != nil {
		t.Fatal(err)
	}
	if err := h.ActivateView("firefox.desktop", "tab:42"); err != nil {
		t.Fatal(err)
	}
	if s.provider != "browser:firefox" || s.command["tabId"] != int64(42) {
		t.Fatalf("provider=%s command=%#v", s.provider, s.command)
	}
}
