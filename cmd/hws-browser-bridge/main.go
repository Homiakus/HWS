package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/providers/nativeframe"
	"github.com/Homiakus/HWS/internal/providers/wire"
)

type ack struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type relay struct {
	mu     sync.Mutex
	conn   net.Conn
	outMu  sync.Mutex
	out    io.Writer
	closed chan struct{}
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		_ = writeNative(os.Stdout, ack{Error: err.Error()})
	}
}

func run(in io.Reader, out io.Writer) error {
	r := &relay{out: out, closed: make(chan struct{})}
	defer r.close()
	for {
		p, err := nativeframe.Read(in)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var raw map[string]any
		if err := json.Unmarshal(p, &raw); err != nil {
			_ = r.writeNative(ack{Error: "invalid json"})
			continue
		}
		e := wire.Envelope{Schema: wire.SchemaVersion, Type: "browser.snapshot", Source: "native-messaging", ReceivedAt: time.Now().UTC(), Payload: append([]byte(nil), p...)}
		if err := r.forward(e); err != nil {
			_ = r.writeNative(ack{Error: err.Error()})
			continue
		}
		if err := r.writeNative(ack{OK: true}); err != nil {
			return err
		}
	}
}

func (r *relay) forward(e wire.Envelope) error {
	conn, err := r.ensureConn()
	if err != nil {
		return err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
	if err := json.NewEncoder(conn).Encode(e); err != nil {
		r.dropConn(conn)
		return err
	}
	return nil
}

func (r *relay) ensureConn() (net.Conn, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conn != nil {
		return r.conn, nil
	}
	path := providerSocketPath()
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("connect provider socket: %w", err)
	}
	r.conn = conn
	go r.readCommands(conn)
	return conn, nil
}

func (r *relay) readCommands(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 16*1024), 256*1024)
	for scanner.Scan() {
		var message map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &message); err != nil {
			continue
		}
		_ = r.writeNative(message)
	}
	r.dropConn(conn)
}

func (r *relay) dropConn(conn net.Conn) {
	r.mu.Lock()
	if r.conn == conn {
		r.conn = nil
		_ = conn.Close()
	}
	r.mu.Unlock()
}

func (r *relay) close() {
	select {
	case <-r.closed:
		return
	default:
		close(r.closed)
	}
	r.mu.Lock()
	if r.conn != nil {
		_ = r.conn.Close()
		r.conn = nil
	}
	r.mu.Unlock()
}

func (r *relay) writeNative(v any) error {
	r.outMu.Lock()
	defer r.outMu.Unlock()
	return writeNative(r.out, v)
}

func writeNative(out io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return nativeframe.Write(out, b)
}

func providerSocketPath() string {
	if path := os.Getenv("HWS_PROVIDER_SOCKET"); path != "" {
		return path
	}
	runtime := os.Getenv("XDG_RUNTIME_DIR")
	if runtime == "" {
		runtime = os.TempDir()
	}
	return filepath.Join(runtime, "hws", "providers.sock")
}
