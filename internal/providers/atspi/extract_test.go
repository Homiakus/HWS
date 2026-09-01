package atspi

import "testing"

func TestExtractViews(t *testing.T) {
	root := Node{Role: "page_tab_list", Children: []Node{{Role: "page_tab", Name: "HWS", States: map[string]bool{"selected": true}}, {Role: "page_tab", Name: "Docs"}}}
	v := ExtractViews(root, "atspi", "w")
	if len(v) != 2 || !v[0].Active || v[0].Title != "HWS" {
		t.Fatalf("views=%#v", v)
	}
}
