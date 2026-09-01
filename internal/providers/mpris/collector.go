package mpris

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
	busPrefix       = "org.mpris.MediaPlayer2."
	objectPath      = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	rootInterface   = "org.mpris.MediaPlayer2"
	playerInterface = "org.mpris.MediaPlayer2.Player"
	propertiesIface = "org.freedesktop.DBus.Properties"
)

type Sink interface {
	Ingest(providers.Snapshot) error
}

type Collector struct {
	conn *dbus.Conn
	sink Sink
	now  func() time.Time

	mu        sync.Mutex
	revisions map[string]uint64
}

func OpenSession(sink Sink) (*Collector, error) {
	if sink == nil {
		return nil, fmt.Errorf("mpris collector: sink is required")
	}
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("mpris collector: connect session bus: %w", err)
	}
	return &Collector{conn: conn, sink: sink, now: time.Now, revisions: map[string]uint64{}}, nil
}

func (c *Collector) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Collector) Run(ctx context.Context, interval time.Duration, report func(error)) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	poll := func() {
		if err := c.Poll(ctx); err != nil && report != nil {
			report(err)
		}
	}
	poll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

func (c *Collector) Poll(ctx context.Context) error {
	bus := c.conn.Object("org.freedesktop.DBus", dbus.ObjectPath("/org/freedesktop/DBus"))
	var names []string
	if err := bus.CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return fmt.Errorf("mpris collector: list names: %w", err)
	}
	for _, name := range names {
		if !strings.HasPrefix(name, busPrefix) {
			continue
		}
		snapshot, err := c.readPlayer(ctx, name)
		if err != nil {
			// Players can disappear between ListNames and property reads.
			continue
		}
		if err := c.sink.Ingest(snapshot); err != nil {
			return fmt.Errorf("mpris collector: ingest %s: %w", name, err)
		}
	}
	return nil
}

func (c *Collector) readPlayer(ctx context.Context, busName string) (providers.Snapshot, error) {
	obj := c.conn.Object(busName, objectPath)
	root, err := getAll(ctx, obj, rootInterface)
	if err != nil {
		return providers.Snapshot{}, err
	}
	player, err := getAll(ctx, obj, playerInterface)
	if err != nil {
		return providers.Snapshot{}, err
	}
	return snapshotFromProperties(busName, root, player, c.nextRevision(busName), c.now()), nil
}

func getAll(ctx context.Context, obj dbus.BusObject, iface string) (map[string]dbus.Variant, error) {
	var props map[string]dbus.Variant
	call := obj.CallWithContext(ctx, propertiesIface+".GetAll", 0, iface)
	if err := call.Store(&props); err != nil {
		return nil, err
	}
	return props, nil
}

func (c *Collector) nextRevision(busName string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.revisions[busName]++
	return c.revisions[busName]
}

func snapshotFromProperties(busName string, root, player map[string]dbus.Variant, revision uint64, now time.Time) providers.Snapshot {
	desktopEntry := variantString(root, "DesktopEntry")
	if desktopEntry == "" {
		desktopEntry = fallbackDesktopEntry(busName)
	}
	if desktopEntry != "" && !strings.HasSuffix(desktopEntry, ".desktop") {
		desktopEntry += ".desktop"
	}
	identity := variantString(root, "Identity")
	if identity == "" {
		identity = strings.TrimSuffix(desktopEntry, ".desktop")
	}
	metadata := variantMap(player, "Metadata")
	artist := ""
	if artists, ok := metadata["xesam:artist"]; ok {
		artist = strings.Join(variantStrings(artists), ", ")
	}
	item := statusprovider.MPRIS{
		AppID:          surface.ApplicationID(desktopEntry),
		PlaybackStatus: variantString(player, "PlaybackStatus"),
		Title:          variantString(metadata, "xesam:title"),
		Artist:         artist,
		PositionMicros: variantInt64(player, "Position"),
		LengthMicros:   variantInt64(metadata, "mpris:length"),
		Revision:       revision,
	}
	snapshot := statusprovider.FromMPRIS(item, now)
	snapshot.ProviderID = "mpris:" + busName
	snapshot.AllowOrphan = true
	snapshot.ApplicationName = identity
	return snapshot
}

func fallbackDesktopEntry(busName string) string {
	value := strings.TrimPrefix(busName, busPrefix)
	if index := strings.IndexByte(value, '.'); index >= 0 {
		value = value[:index]
	}
	return strings.TrimSpace(value)
}

func variantString(values map[string]dbus.Variant, key string) string {
	variant, ok := values[key]
	if !ok {
		return ""
	}
	value, _ := variant.Value().(string)
	return value
}

func variantInt64(values map[string]dbus.Variant, key string) int64 {
	variant, ok := values[key]
	if !ok {
		return 0
	}
	value, _ := variant.Value().(int64)
	return value
}

func variantMap(values map[string]dbus.Variant, key string) map[string]dbus.Variant {
	variant, ok := values[key]
	if !ok {
		return nil
	}
	value, _ := variant.Value().(map[string]dbus.Variant)
	return value
}

func variantStrings(variant dbus.Variant) []string {
	value, _ := variant.Value().([]string)
	return value
}
