package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Homiakus/HWS/internal/providers/nativeframe"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

type envelope struct {
	Schema     uint32          `json:"schema"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	ReceivedAt time.Time       `json:"receivedAt"`
	Payload    json.RawMessage `json:"payload"`
}
type ack struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		_ = nativeframe.Write(os.Stdout, mustJSON(ack{Error: err.Error()}))
	}
}
func run(in io.Reader, out io.Writer) error {
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
			_ = nativeframe.Write(out, mustJSON(ack{Error: "invalid json"}))
			continue
		}
		e := envelope{Schema: 1, Type: "browser.snapshot", Source: "native-messaging", ReceivedAt: time.Now().UTC(), Payload: append([]byte(nil), p...)}
		if err := forward(e); err != nil {
			_ = nativeframe.Write(out, mustJSON(ack{Error: err.Error()}))
			continue
		}
		if err := nativeframe.Write(out, mustJSON(ack{OK: true})); err != nil {
			return err
		}
	}
}
func forward(e envelope) error {
	path := os.Getenv("HWS_PROVIDER_SOCKET")
	if path == "" {
		runtime := os.Getenv("XDG_RUNTIME_DIR")
		if runtime == "" {
			runtime = os.TempDir()
		}
		path = filepath.Join(runtime, "hws", "providers.sock")
	}
	c, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err != nil {
		return fmt.Errorf("connect provider socket: %w", err)
	}
	defer c.Close()
	_ = c.SetWriteDeadline(time.Now().Add(time.Second))
	enc := json.NewEncoder(c)
	return enc.Encode(e)
}
func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }
