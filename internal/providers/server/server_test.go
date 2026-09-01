package server

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/providers/browser"
	"github.com/Homiakus/HWS/internal/providers/wire"
)

type sink struct{ ch chan providers.Snapshot }

func (s *sink) Ingest(x providers.Snapshot) error { s.ch <- x; return nil }

func TestServerIngestsBrowserSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "providers.sock")
	sk := &sink{ch: make(chan providers.Snapshot, 1)}
	srv := New(path, sk)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx) }()
	deadline := time.Now().Add(time.Second)
	for {
		c, err := net.Dial("unix", path)
		if err == nil {
			payload, _ := json.Marshal(browser.Snapshot{Schema: 1, Browser: "firefox", AppID: "firefox.desktop", Revision: 1, CapturedAt: time.Now(), Tabs: []browser.Tab{{ID: 1, WindowID: 1, Title: "HWS", Active: true}}})
			env, _ := json.Marshal(wire.Envelope{Schema: 1, Type: "browser.snapshot", Source: "test", ReceivedAt: time.Now(), Payload: payload})
			_, _ = c.Write(append(env, '\n'))
			_ = c.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case got := <-sk.ch:
		if got.AppID != "firefox.desktop" || len(got.Windows) != 1 {
			t.Fatalf("snapshot=%#v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("snapshot not received")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}
