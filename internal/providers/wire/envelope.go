package wire

import (
	"encoding/json"
	"fmt"
	"time"
)

const SchemaVersion uint32 = 1
const MaxEnvelopeBytes = 4 << 20

type Envelope struct {
	Schema     uint32          `json:"schema"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	ReceivedAt time.Time       `json:"receivedAt"`
	Payload    json.RawMessage `json:"payload"`
}

func (e Envelope) Validate() error {
	if e.Schema != SchemaVersion {
		return fmt.Errorf("provider schema %d unsupported", e.Schema)
	}
	if e.Type == "" || e.Source == "" {
		return fmt.Errorf("provider type and source are required")
	}
	if len(e.Payload) == 0 || len(e.Payload) > MaxEnvelopeBytes {
		return fmt.Errorf("provider payload size %d out of range", len(e.Payload))
	}
	return nil
}
