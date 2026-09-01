package atspi

import (
	"fmt"
	"strings"

	"github.com/Homiakus/HWS/internal/surface"
)

type Node struct {
	Role       string
	Name       string
	States     map[string]bool
	Attributes map[string]string
	Children   []Node
}

func ExtractViews(root Node, providerID, providerWindowID string) []surface.View {
	var out []surface.View
	var walk func(Node, string)
	walk = func(n Node, path string) {
		role := strings.ToLower(strings.ReplaceAll(n.Role, "-", "_"))
		if role == "page_tab" || role == "tab" {
			id := ""
			if n.Attributes != nil {
				id = n.Attributes["id"]
			}
			if id == "" {
				id = fmt.Sprintf("%s:%s", providerWindowID, path)
			}
			kind := surface.ViewTab
			dirty := false
			if n.Attributes != nil {
				dirty = n.Attributes["dirty"] == "true"
			}
			out = append(out, surface.View{ID: surface.ViewID(id), ProviderID: providerID, ProviderWindowID: providerWindowID, Kind: kind, Title: n.Name, Active: n.States["active"] || n.States["selected"], Dirty: dirty, Attention: surface.AttentionNormal})
		}
		for i, c := range n.Children {
			walk(c, fmt.Sprintf("%s/%d", path, i))
		}
	}
	walk(root, "0")
	return out
}
