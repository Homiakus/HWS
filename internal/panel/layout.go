package panel

import (
	"cmp"
	"slices"
)

type Card struct {
	ID             string
	Priority       int
	Urgent         bool
	MinWidth       int
	PreferredWidth int
	MaxWidth       int
}

type Placement struct {
	ID      string  `json:"id"`
	Density Density `json:"density"`
	Width   int     `json:"width"`
}

type LayoutResult struct {
	Visible   []Placement `json:"visible"`
	Overflow  []string    `json:"overflow"`
	UsedWidth int         `json:"usedWidth"`
	Overfull  bool        `json:"overfull"`
}

func Layout(available, gap int, cards []Card) LayoutResult {
	if available < 0 {
		available = 0
	}
	if gap < 0 {
		gap = 0
	}
	work := append([]Card(nil), cards...)
	slices.SortFunc(work, func(a, b Card) int {
		if a.Priority != b.Priority {
			return cmp.Compare(b.Priority, a.Priority)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	placements := make([]Placement, len(work))
	for i, c := range work {
		min, pref, max := normalizeWidths(c)
		_ = min
		w := pref
		if w > max {
			w = max
		}
		placements[i] = Placement{ID: c.ID, Density: DensityFull, Width: w}
	}
	fits := func(ps []Placement) bool { return used(ps, gap) <= available }
	collapseOrder := make([]int, len(work))
	for i := range work {
		collapseOrder[i] = i
	}
	slices.SortFunc(collapseOrder, func(i, j int) int {
		a, b := work[i], work[j]
		if a.Urgent != b.Urgent {
			if !a.Urgent {
				return -1
			}
			return 1
		}
		if a.Priority != b.Priority {
			return cmp.Compare(a.Priority, b.Priority)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	for _, idx := range collapseOrder {
		if fits(placements) {
			break
		}
		min, pref, _ := normalizeWidths(work[idx])
		compact := min + (pref-min)/2
		if compact < min {
			compact = min
		}
		placements[idx].Density = DensityCompact
		placements[idx].Width = compact
	}
	for _, idx := range collapseOrder {
		if fits(placements) {
			break
		}
		min, _, _ := normalizeWidths(work[idx])
		placements[idx].Density = DensityMicro
		placements[idx].Width = min
	}
	visible := append([]Placement(nil), placements...)
	overflow := []string{}
	for _, idx := range collapseOrder {
		if used(visible, gap) <= available {
			break
		}
		if work[idx].Urgent {
			continue
		}
		id := work[idx].ID
		for i := range visible {
			if visible[i].ID == id {
				visible = append(visible[:i], visible[i+1:]...)
				overflow = append(overflow, id)
				break
			}
		}
	}
	slices.SortFunc(visible, func(a, b Placement) int {
		ai, bi := indexOf(work, a.ID), indexOf(work, b.ID)
		return cmp.Compare(ai, bi)
	})
	return LayoutResult{Visible: visible, Overflow: overflow, UsedWidth: used(visible, gap), Overfull: used(visible, gap) > available}
}

func normalizeWidths(c Card) (int, int, int) {
	min := c.MinWidth
	if min <= 0 {
		min = 48
	}
	pref := c.PreferredWidth
	if pref < min {
		pref = min
	}
	max := c.MaxWidth
	if max < pref {
		max = pref
	}
	return min, pref, max
}
func used(ps []Placement, gap int) int {
	n := 0
	for _, p := range ps {
		n += p.Width
	}
	if len(ps) > 1 {
		n += gap * (len(ps) - 1)
	}
	return n
}
func indexOf(cards []Card, id string) int {
	for i, c := range cards {
		if c.ID == id {
			return i
		}
	}
	return len(cards)
}
