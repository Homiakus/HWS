package daemon

import "github.com/Homiakus/HWS/internal/surface"

func (h *Hub) SurfaceSnapshot() surface.Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.surfaceSnapshot.Clone()
}
