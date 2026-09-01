package panel

import "fmt"

type Edge string

const (
	EdgeTop    Edge = "top"
	EdgeBottom Edge = "bottom"
	EdgeLeft   Edge = "left"
	EdgeRight  Edge = "right"
)

type OverflowMode string

const (
	OverflowPopover OverflowMode = "popover"
	OverflowScroll  OverflowMode = "scroll"
	OverflowNone    OverflowMode = "none"
)

type Density string

const (
	DensityAdaptive Density = "adaptive"
	DensityFull     Density = "full"
	DensityCompact  Density = "compact"
	DensityMicro    Density = "micro"
)

type SurfaceConfig struct {
	Mode       string `json:"mode"`
	MaxVisible int    `json:"maxVisible"`
	Overflow   string `json:"overflow"`
}

type AppConfig struct {
	Density        Density       `json:"density"`
	MinWidth       int           `json:"minWidth"`
	PreferredWidth int           `json:"preferredWidth"`
	MaxWidth       int           `json:"maxWidth"`
	Surfaces       SurfaceConfig `json:"surfaces"`
}

type Widget struct {
	ID      string `json:"id"`
	Variant string `json:"variant,omitempty"`
	Format  string `json:"format,omitempty"`
}

type Group struct {
	ID           string            `json:"id"`
	Source       string            `json:"source,omitempty"`
	App          *AppConfig        `json:"app,omitempty"`
	Widgets      []Widget          `json:"widgets,omitempty"`
	Interactions map[string]string `json:"interactions,omitempty"`
}

type Spec struct {
	ID       string       `json:"id"`
	Edge     Edge         `json:"edge"`
	Height   int          `json:"height"`
	Gap      int          `json:"gap"`
	Overflow OverflowMode `json:"overflow"`
	Groups   []Group      `json:"groups"`
}

func DefaultSpec(id string) Spec {
	return Spec{ID: id, Edge: EdgeTop, Height: 40, Gap: 6, Overflow: OverflowPopover}
}

func (s Spec) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("panel id is required")
	}
	switch s.Edge {
	case EdgeTop, EdgeBottom, EdgeLeft, EdgeRight:
	default:
		return fmt.Errorf("invalid panel edge %q", s.Edge)
	}
	if s.Height < 24 || s.Height > 160 {
		return fmt.Errorf("panel height must be 24..160")
	}
	if s.Gap < 0 || s.Gap > 64 {
		return fmt.Errorf("panel gap must be 0..64")
	}
	switch s.Overflow {
	case OverflowPopover, OverflowScroll, OverflowNone:
	default:
		return fmt.Errorf("invalid overflow %q", s.Overflow)
	}
	for _, g := range s.Groups {
		if g.ID == "" {
			return fmt.Errorf("group id is required")
		}
		if g.App != nil {
			a := g.App
			if a.MinWidth <= 0 || a.PreferredWidth < a.MinWidth || a.MaxWidth < a.PreferredWidth {
				return fmt.Errorf("group %s: invalid app width constraints", g.ID)
			}
			if a.Surfaces.MaxVisible < 0 || a.Surfaces.MaxVisible > 20 {
				return fmt.Errorf("group %s: surfaces.max_visible must be 0..20", g.ID)
			}
		}
	}
	return nil
}
