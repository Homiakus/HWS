package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	browserprovider "github.com/Homiakus/HWS/internal/providers/browser"
	vscodeprovider "github.com/Homiakus/HWS/internal/providers/vscode"
	"github.com/Homiakus/HWS/internal/providers/wire"
)

type Sink interface {
	Ingest(providers.Snapshot) error
}

type Server struct {
	Path     string
	Sink     Sink
	mu       sync.Mutex
	listener net.Listener
}

func New(path string, sink Sink) *Server { return &Server{Path: path, Sink: sink} }

func (s *Server) Serve(ctx context.Context) error {
	if s.Sink == nil {
		return fmt.Errorf("provider server: sink is required")
	}
	if s.Path == "" {
		return fmt.Errorf("provider server: socket path is required")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	if err := removeOwnedSocket(s.Path); err != nil {
		return err
	}
	ln, err := net.Listen("unix", s.Path)
	if err != nil {
		return err
	}
	if err := os.Chmod(s.Path, 0o600); err != nil {
		_ = ln.Close()
		return err
	}
	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	defer func() { _ = ln.Close(); _ = os.Remove(s.Path) }()
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}
		go s.handle(ctx, conn)
	}
}

func (s *Server) handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, wire.MaxEnvelopeBytes)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		var env wire.Envelope
		if err := json.Unmarshal(scanner.Bytes(), &env); err != nil {
			continue
		}
		if err := env.Validate(); err != nil {
			continue
		}
		_ = s.dispatch(env)
	}
}

func (s *Server) dispatch(env wire.Envelope) error {
	switch env.Type {
	case "browser.snapshot":
		var x browserprovider.Snapshot
		if err := json.Unmarshal(env.Payload, &x); err != nil {
			return err
		}
		if x.CapturedAt.IsZero() {
			x.CapturedAt = env.ReceivedAt
		}
		return s.Sink.Ingest(x.ProviderSnapshot())
	case "vscode.snapshot":
		var x vscodeprovider.Snapshot
		if err := json.Unmarshal(env.Payload, &x); err != nil {
			return err
		}
		if x.CapturedAt.IsZero() {
			x.CapturedAt = env.ReceivedAt
		}
		return s.Sink.Ingest(x.ProviderSnapshot())
	default:
		return fmt.Errorf("provider message type %q unsupported", env.Type)
	}
}

func removeOwnedSocket(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("provider socket path exists and is not a socket: %s", path)
	}
	return os.Remove(path)
}
