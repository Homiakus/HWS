package shellaction

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestBrokerRoundTrip(t *testing.T) {
	broker := NewBroker(time.Second)
	broker.SetEmitter(func(request Request) error {
		return broker.Complete(Result{Schema: SchemaVersion, ID: request.ID, Success: true, Changed: true})
	})
	result, err := broker.Request(context.Background(), Request{
		Kind:         KindEnsureDesktopApp,
		WorkspaceID:  "dev",
		ResourceID:   "editor",
		DesktopAppID: "dev.zed.Zed.desktop",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || !result.Changed {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestBrokerFailsClosedWithoutExecutor(t *testing.T) {
	broker := NewBroker(10 * time.Millisecond)
	_, err := broker.Request(context.Background(), Request{
		Kind:         KindEnsureDesktopApp,
		WorkspaceID:  "dev",
		ResourceID:   "editor",
		DesktopAppID: "dev.zed.Zed.desktop",
	})
	if err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrokerTimesOutAndRejectsLateCompletion(t *testing.T) {
	broker := NewBroker(10 * time.Millisecond)
	var requestID string
	broker.SetEmitter(func(request Request) error {
		requestID = request.ID
		return nil
	})
	_, err := broker.Request(context.Background(), Request{
		Kind:         KindEnsureDesktopApp,
		WorkspaceID:  "dev",
		ResourceID:   "editor",
		DesktopAppID: "dev.zed.Zed.desktop",
	})
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if err := broker.Complete(Result{Schema: SchemaVersion, ID: requestID, Success: true}); err == nil {
		t.Fatal("late completion accepted")
	}
}

func TestDecodeResultIsStrict(t *testing.T) {
	if _, err := DecodeResult([]byte(`{"schema":1,"id":"x","success":true,"exec":"no"}`)); err == nil {
		t.Fatal("unknown result field accepted")
	}
}
