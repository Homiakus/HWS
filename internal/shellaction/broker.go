package shellaction

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const SchemaVersion uint32 = 1

type Kind string

const (
	KindEnsureDesktopApp Kind = "ensure_desktop_app"
	KindCloseWindow      Kind = "close_window"
)

type Request struct {
	Schema       uint32 `json:"schema"`
	ID           string `json:"id"`
	Kind         Kind   `json:"kind"`
	WorkspaceID  string `json:"workspaceId"`
	ResourceID   string `json:"resourceId"`
	DesktopAppID string `json:"desktopAppId,omitempty"`
	WindowID     string `json:"windowId,omitempty"`
}

type Result struct {
	Schema  uint32 `json:"schema"`
	ID      string `json:"id"`
	Success bool   `json:"success"`
	Changed bool   `json:"changed,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type Emitter func(Request) error

type Broker struct {
	mu      sync.Mutex
	emitter Emitter
	pending map[string]chan Result
	timeout time.Duration
}

func NewBroker(timeout time.Duration) *Broker {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &Broker{pending: make(map[string]chan Result), timeout: timeout}
}

func (b *Broker) SetEmitter(emitter Emitter) {
	b.mu.Lock()
	b.emitter = emitter
	b.mu.Unlock()
}

func (b *Broker) Request(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	request.Schema = SchemaVersion
	if strings.TrimSpace(request.ID) == "" {
		request.ID = randomID()
	}
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}

	resultCh := make(chan Result, 1)
	b.mu.Lock()
	emitter := b.emitter
	if emitter == nil {
		b.mu.Unlock()
		return Result{}, errors.New("shell action executor is unavailable")
	}
	if _, exists := b.pending[request.ID]; exists {
		b.mu.Unlock()
		return Result{}, fmt.Errorf("shell action %s is already pending", request.ID)
	}
	b.pending[request.ID] = resultCh
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.pending, request.ID)
		b.mu.Unlock()
	}()

	if err := emitter(request); err != nil {
		return Result{}, fmt.Errorf("emit shell action %s: %w", request.ID, err)
	}

	timer := time.NewTimer(b.timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	case <-timer.C:
		return Result{}, fmt.Errorf("shell action %s timed out", request.ID)
	case result := <-resultCh:
		return result, nil
	}
}

func (b *Broker) Complete(result Result) error {
	if result.Schema != SchemaVersion {
		return fmt.Errorf("shell action result: unsupported schema %d", result.Schema)
	}
	if strings.TrimSpace(result.ID) == "" {
		return errors.New("shell action result: id is required")
	}
	b.mu.Lock()
	resultCh := b.pending[result.ID]
	b.mu.Unlock()
	if resultCh == nil {
		return fmt.Errorf("shell action result %s has no pending request", result.ID)
	}
	select {
	case resultCh <- result:
		return nil
	default:
		return fmt.Errorf("shell action result %s was already completed", result.ID)
	}
}

func (b *Broker) CompleteJSON(payload string) error {
	result, err := DecodeResult([]byte(payload))
	if err != nil {
		return err
	}
	return b.Complete(result)
}

func EncodeRequest(request Request) (string, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func DecodeResult(data []byte) (Result, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result Result
	if err := decoder.Decode(&result); err != nil {
		return Result{}, fmt.Errorf("shell action result: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return Result{}, errors.New("shell action result: trailing JSON value")
		}
		return Result{}, fmt.Errorf("shell action result: trailing data: %w", err)
	}
	if result.Schema != SchemaVersion {
		return Result{}, fmt.Errorf("shell action result: unsupported schema %d", result.Schema)
	}
	if strings.TrimSpace(result.ID) == "" {
		return Result{}, errors.New("shell action result: id is required")
	}
	return result, nil
}

func validateRequest(request Request) error {
	if request.Schema != SchemaVersion {
		return fmt.Errorf("shell action request: unsupported schema %d", request.Schema)
	}
	if strings.TrimSpace(request.ID) == "" || strings.TrimSpace(request.WorkspaceID) == "" || strings.TrimSpace(request.ResourceID) == "" {
		return errors.New("shell action request: id, workspace id and resource id are required")
	}
	switch request.Kind {
	case KindEnsureDesktopApp:
		if strings.TrimSpace(request.DesktopAppID) == "" {
			return errors.New("shell action request: desktop app id is required")
		}
	case KindCloseWindow:
		if strings.TrimSpace(request.WindowID) == "" {
			return errors.New("shell action request: window id is required")
		}
	default:
		return fmt.Errorf("shell action request: unsupported kind %q", request.Kind)
	}
	return nil
}

func randomID() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("shell-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(data[:])
}
