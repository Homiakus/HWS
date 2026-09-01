package panel

import "testing"

func TestLayoutCollapsesThenOverflowsLowPriority(t *testing.T) {
	cards := []Card{{ID: "zed", Priority: 100, MinWidth: 64, PreferredWidth: 160, MaxWidth: 220}, {ID: "firefox", Priority: 80, MinWidth: 64, PreferredWidth: 160, MaxWidth: 220}, {ID: "music", Priority: 10, MinWidth: 48, PreferredWidth: 120, MaxWidth: 160}}
	r := Layout(150, 6, cards)
	if len(r.Overflow) == 0 || r.Overflow[0] != "music" {
		t.Fatalf("overflow=%v", r.Overflow)
	}
	if r.Overfull {
		t.Fatalf("unexpected overfull: %#v", r)
	}
}
func TestLayoutNeverOverflowsUrgentCard(t *testing.T) {
	r := Layout(30, 6, []Card{{ID: "urgent", Priority: 1, Urgent: true, MinWidth: 64, PreferredWidth: 100, MaxWidth: 100}})
	if len(r.Overflow) != 0 || !r.Overfull {
		t.Fatalf("result=%#v", r)
	}
}
