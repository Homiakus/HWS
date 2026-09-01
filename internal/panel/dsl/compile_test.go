package dsl

import "testing"

const sample = `
panel "main" {
 edge = "top"
 height = 40
 gap = 6
 overflow = "popover"
 group "applications" {
   source = "running"
   app {
     density = "adaptive"
     min_width = 64
     preferred_width = 156
     max_width = 240
     surfaces { mode = "segments" max_visible = 4 overflow = "count" }
   }
   on "click" { action = "focus_or_cycle" }
 }
 group "system" { widget "clock" { format = "HH:mm" } }
}`

func TestCompile(t *testing.T) {
	s, err := Compile([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "main" || len(s.Groups) != 2 || s.Groups[0].App == nil || s.Groups[0].App.Surfaces.MaxVisible != 4 {
		t.Fatalf("spec=%#v", s)
	}
}
func TestManagerKeepsLastKnownGood(t *testing.T) {
	var m Manager
	s, rev, err := m.Apply([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if _, rev2, err := m.Apply([]byte(`panel "bad" { heigth = 40 }`)); err == nil || rev2 != rev {
		t.Fatalf("revision=%d err=%v", rev2, err)
	}
	cur, _, ok := m.Current()
	if !ok || cur.ID != s.ID {
		t.Fatalf("current=%#v", cur)
	}
}
