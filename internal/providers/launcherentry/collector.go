package launcherentry

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/providers"
	statusprovider "github.com/Homiakus/HWS/internal/providers/status"
	"github.com/Homiakus/HWS/internal/surface"
	"github.com/godbus/dbus/v5"
)

const (
	interfaceName = "com.canonical.Unity.LauncherEntry"
	memberUpdate  = "Update"
)

type Sink interface {
	Ingest(providers.Snapshot) error
}

type entryState struct {
	AppID surface.ApplicationID

	Count           int64
	CountVisible    bool
	Progress        float64
	ProgressVisible bool
	Urgent          bool
	Updating        bool
	Revision        uint64
}

type Collector struct {
	conn *dbus.Conn
	sink Sink
	now  func() time.Time

	mu      sync.Mutex
	entries map[surface.ApplicationID]entryState
}

func OpenSession(sink Sink) (*Collector, error) {
	if sink == nil {
		return nil, fmt.Errorf("launcher entry collector: sink is required")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("launcher entry collector: connect session bus: %w", err)
	}
	return &Collector{conn: conn, sink: sink, now: time.Now, entries: map[surface.ApplicationID]entryState{}}, nil
}

func (c *Collector) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Collector) Run(ctx context.Context, report func(error)) {
	signals := make(chan *dbus.Signal, 64)
	c.conn.Signal(signals)
	defer c.conn.RemoveSignal(signals)
	if err := c.conn.AddMatchSignal(
		dbus.WithMatchInterface(interfaceName),
		dbus.WithMatchMember(memberUpdate),
	); err != nil {
		if report != nil {
			report(fmt.Errorf("launcher entry collector: add match: %w", err))
		}
		return
	}

	// LauncherEntry updates are deltas. Re-publishing the merged state keeps the
	// registry TTL fresh without changing semantic surface revisions.
	refresh := time.NewTicker(10 * time.Second)
	defer refresh.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case signal := <-signals:
			if signal == nil || signal.Name != interfaceName+"."+memberUpdate {
				continue
			}
			if err := c.HandleSignal(signal); err != nil && report != nil {
				report(err)
			}
		case <-refresh.C:
			if err := c.refresh(); err != nil && report != nil {
				report(err)
			}
		}
	}
}

func (c *Collector) HandleSignal(signal *dbus.Signal) error {
	if signal == nil || len(signal.Body) != 2 {
		return fmt.Errorf("launcher entry collector: invalid Update body")
	}
	appURI, ok := signal.Body[0].(string)
	if !ok {
		return fmt.Errorf("launcher entry collector: invalid app URI")
	}
	properties, ok := signal.Body[1].(map[string]dbus.Variant)
	if !ok {
		return fmt.Errorf("launcher entry collector: invalid properties map")
	}
	appID, err := appIDFromURI(appURI)
	if err != nil {
		return err
	}

	c.mu.Lock()
	state := c.entries[appID]
	state.AppID = appID
	applyProperties(&state, properties)
	state.Revision++
	c.entries[appID] = state
	c.mu.Unlock()

	return c.ingest(state, c.now())
}

func (c *Collector) refresh() error {
	c.mu.Lock()
	states := make([]entryState, 0, len(c.entries))
	for _, state := range c.entries {
		states = append(states, state)
	}
	c.mu.Unlock()
	for _, state := range states {
		if err := c.ingest(state, c.now()); err != nil {
			return err
		}
	}
	return nil
}

func (c *Collector) ingest(state entryState, now time.Time) error {
	var count *int64
	if state.CountVisible {
		value := state.Count
		count = &value
	}
	var progress *float64
	if state.ProgressVisible {
		value := state.Progress
		progress = &value
	}
	snapshot := statusprovider.FromLauncherEntry(statusprovider.LauncherEntry{
		AppID:    state.AppID,
		Progress: progress,
		Count:    count,
		Urgent:   state.Urgent,
		Updating: state.Updating,
		Revision: state.Revision,
	}, now)
	snapshot.ProviderID = "launcher-entry"
	snapshot.AllowOrphan = true
	snapshot.ApplicationName = strings.TrimSuffix(string(state.AppID), ".desktop")
	snapshot.TTL = 30 * time.Second
	return c.sink.Ingest(snapshot)
}

func appIDFromURI(uri string) (surface.ApplicationID, error) {
	const prefix = "application://"
	if !strings.HasPrefix(uri, prefix) {
		return "", fmt.Errorf("launcher entry collector: unsupported app URI %q", uri)
	}
	id := strings.TrimSpace(strings.TrimPrefix(uri, prefix))
	if id == "" || strings.ContainsAny(id, "/\\") {
		return "", fmt.Errorf("launcher entry collector: invalid desktop id %q", id)
	}
	if !strings.HasSuffix(id, ".desktop") {
		id += ".desktop"
	}
	return surface.ApplicationID(id), nil
}

func applyProperties(state *entryState, properties map[string]dbus.Variant) {
	if value, ok := propertyInt64(properties, "count"); ok {
		state.Count = value
	}
	if value, ok := propertyBool(properties, "count-visible"); ok {
		state.CountVisible = value
	}
	if value, ok := propertyFloat64(properties, "progress"); ok {
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		state.Progress = value
	}
	if value, ok := propertyBool(properties, "progress-visible"); ok {
		state.ProgressVisible = value
	}
	if value, ok := propertyBool(properties, "urgent"); ok {
		state.Urgent = value
	}
	if value, ok := propertyBool(properties, "updating"); ok {
		state.Updating = value
	}
}

func propertyBool(properties map[string]dbus.Variant, key string) (bool, bool) {
	variant, ok := properties[key]
	if !ok {
		return false, false
	}
	value, ok := variant.Value().(bool)
	return value, ok
}

func propertyInt64(properties map[string]dbus.Variant, key string) (int64, bool) {
	variant, ok := properties[key]
	if !ok {
		return 0, false
	}
	value, ok := variant.Value().(int64)
	return value, ok
}

func propertyFloat64(properties map[string]dbus.Variant, key string) (float64, bool) {
	variant, ok := properties[key]
	if !ok {
		return 0, false
	}
	value, ok := variant.Value().(float64)
	return value, ok
}
