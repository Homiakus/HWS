package dsl

import (
	"fmt"

	"github.com/Homiakus/HWS/internal/panel"
)

func Compile(src []byte) (panel.Spec, error) {
	blocks, err := Parse(src)
	if err != nil {
		return panel.Spec{}, err
	}
	if len(blocks) != 1 || blocks[0].Type != "panel" || len(blocks[0].Labels) != 1 {
		return panel.Spec{}, fmt.Errorf("dsl: exactly one panel \"id\" block is required")
	}
	root := blocks[0]
	if err := unknownAttrs(root.Attrs, "edge", "height", "gap", "overflow"); err != nil {
		return panel.Spec{}, fmt.Errorf("panel: %w", err)
	}
	s := panel.DefaultSpec(root.Labels[0])
	if v, ok := stringAttr(root, "edge"); ok {
		s.Edge = panel.Edge(v)
	}
	if v, ok := intAttr(root, "height"); ok {
		s.Height = v
	}
	if v, ok := intAttr(root, "gap"); ok {
		s.Gap = v
	}
	if v, ok := stringAttr(root, "overflow"); ok {
		s.Overflow = panel.OverflowMode(v)
	}
	for _, b := range root.Blocks {
		if b.Type != "group" {
			return panel.Spec{}, fmt.Errorf("panel: unsupported block %q", b.Type)
		}
		g, err := compileGroup(b)
		if err != nil {
			return panel.Spec{}, err
		}
		s.Groups = append(s.Groups, g)
	}
	if err := s.Validate(); err != nil {
		return panel.Spec{}, err
	}
	return s, nil
}

func compileGroup(b Block) (panel.Group, error) {
	if len(b.Labels) != 1 {
		return panel.Group{}, fmt.Errorf("group: exactly one label is required")
	}
	if err := unknownAttrs(b.Attrs, "source"); err != nil {
		return panel.Group{}, fmt.Errorf("group %s: %w", b.Labels[0], err)
	}
	g := panel.Group{ID: b.Labels[0], Interactions: map[string]string{}}
	if v, ok := stringAttr(b, "source"); ok {
		g.Source = v
	}
	for _, c := range b.Blocks {
		switch c.Type {
		case "app":
			if g.App != nil {
				return panel.Group{}, fmt.Errorf("group %s: duplicate app block", g.ID)
			}
			a, err := compileApp(c)
			if err != nil {
				return panel.Group{}, fmt.Errorf("group %s: %w", g.ID, err)
			}
			g.App = &a
		case "widget":
			if len(c.Labels) != 1 {
				return panel.Group{}, fmt.Errorf("group %s: widget requires id", g.ID)
			}
			if err := unknownAttrs(c.Attrs, "variant", "format"); err != nil {
				return panel.Group{}, err
			}
			w := panel.Widget{ID: c.Labels[0]}
			w.Variant, _ = stringAttr(c, "variant")
			w.Format, _ = stringAttr(c, "format")
			g.Widgets = append(g.Widgets, w)
		case "on":
			if len(c.Labels) != 1 {
				return panel.Group{}, fmt.Errorf("group %s: on requires event label", g.ID)
			}
			if err := unknownAttrs(c.Attrs, "action"); err != nil {
				return panel.Group{}, err
			}
			a, ok := stringAttr(c, "action")
			if !ok || a == "" {
				return panel.Group{}, fmt.Errorf("group %s: on %s requires action", g.ID, c.Labels[0])
			}
			g.Interactions[c.Labels[0]] = a
		default:
			return panel.Group{}, fmt.Errorf("group %s: unsupported block %q", g.ID, c.Type)
		}
	}
	return g, nil
}
func compileApp(b Block) (panel.AppConfig, error) {
	if len(b.Labels) != 0 {
		return panel.AppConfig{}, fmt.Errorf("app block does not accept labels")
	}
	if err := unknownAttrs(b.Attrs, "density", "min_width", "preferred_width", "max_width"); err != nil {
		return panel.AppConfig{}, err
	}
	a := panel.AppConfig{Density: panel.DensityAdaptive, MinWidth: 64, PreferredWidth: 156, MaxWidth: 240, Surfaces: panel.SurfaceConfig{Mode: "segments", MaxVisible: 4, Overflow: "count"}}
	if v, ok := stringAttr(b, "density"); ok {
		a.Density = panel.Density(v)
	}
	if v, ok := intAttr(b, "min_width"); ok {
		a.MinWidth = v
	}
	if v, ok := intAttr(b, "preferred_width"); ok {
		a.PreferredWidth = v
	}
	if v, ok := intAttr(b, "max_width"); ok {
		a.MaxWidth = v
	}
	for _, c := range b.Blocks {
		if c.Type != "surfaces" {
			return panel.AppConfig{}, fmt.Errorf("app: unsupported block %q", c.Type)
		}
		if err := unknownAttrs(c.Attrs, "mode", "max_visible", "overflow"); err != nil {
			return panel.AppConfig{}, err
		}
		if v, ok := stringAttr(c, "mode"); ok {
			a.Surfaces.Mode = v
		}
		if v, ok := intAttr(c, "max_visible"); ok {
			a.Surfaces.MaxVisible = v
		}
		if v, ok := stringAttr(c, "overflow"); ok {
			a.Surfaces.Overflow = v
		}
	}
	return a, nil
}
func unknownAttrs(m map[string]any, allowed ...string) error {
	set := map[string]bool{}
	for _, a := range allowed {
		set[a] = true
	}
	for k := range m {
		if !set[k] {
			return fmt.Errorf("unknown attribute %q", k)
		}
	}
	return nil
}
func stringAttr(b Block, key string) (string, bool) {
	v, ok := b.Attrs[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
func intAttr(b Block, key string) (int, bool) {
	v, ok := b.Attrs[key]
	if !ok {
		return 0, false
	}
	i, ok := v.(int)
	return i, ok
}
