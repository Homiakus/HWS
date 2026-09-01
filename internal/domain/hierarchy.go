package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

type NodeKind string

const (
	NodeCategory NodeKind = "category"
	NodeProject  NodeKind = "project"
	NodeTask     NodeKind = "task"
	NodeAction   NodeKind = "action"
	NodeWidget   NodeKind = "widget"
	NodeQuery    NodeKind = "query"
)

type Node struct {
	ID          NodeID
	ParentID    NodeID
	Kind        NodeKind
	Title       string
	Order       int
	WorkspaceID WorkspaceID
}

type Tree struct {
	root     NodeID
	nodes    map[NodeID]Node
	children map[NodeID][]NodeID
}

func NewTree(input []Node) (*Tree, error) {
	if len(input) == 0 {
		return nil, errors.New("hierarchy: at least one node is required")
	}

	t := &Tree{
		nodes:    make(map[NodeID]Node, len(input)),
		children: make(map[NodeID][]NodeID, len(input)),
	}

	roots := 0
	for _, node := range input {
		if !validID(node.ID) {
			return nil, errors.New("hierarchy: node id is required")
		}
		if strings.TrimSpace(node.Title) == "" {
			return nil, fmt.Errorf("hierarchy: node %q has empty title", node.ID)
		}
		if _, exists := t.nodes[node.ID]; exists {
			return nil, fmt.Errorf("hierarchy: duplicate node id %q", node.ID)
		}
		if node.Kind == "" {
			node.Kind = NodeCategory
		}
		t.nodes[node.ID] = node
		if node.ParentID == "" {
			t.root = node.ID
			roots++
		}
	}

	if roots != 1 {
		return nil, fmt.Errorf("hierarchy: exactly one root is required, got %d", roots)
	}

	for _, node := range t.nodes {
		if node.ParentID == "" {
			continue
		}
		if _, ok := t.nodes[node.ParentID]; !ok {
			return nil, fmt.Errorf("hierarchy: node %q references missing parent %q", node.ID, node.ParentID)
		}
		t.children[node.ParentID] = append(t.children[node.ParentID], node.ID)
	}

	for id := range t.nodes {
		if err := t.validatePathToRoot(id); err != nil {
			return nil, err
		}
	}

	for parent, ids := range t.children {
		sort.SliceStable(ids, func(i, j int) bool {
			left := t.nodes[ids[i]]
			right := t.nodes[ids[j]]
			if left.Order != right.Order {
				return left.Order < right.Order
			}
			if left.Title != right.Title {
				return left.Title < right.Title
			}
			return left.ID < right.ID
		})
		t.children[parent] = ids
	}

	return t, nil
}

func (t *Tree) validatePathToRoot(start NodeID) error {
	seen := make(map[NodeID]struct{})
	current := start
	for {
		if _, duplicate := seen[current]; duplicate {
			return fmt.Errorf("hierarchy: cycle detected at node %q", current)
		}
		seen[current] = struct{}{}

		node := t.nodes[current]
		if node.ParentID == "" {
			if current != t.root {
				return fmt.Errorf("hierarchy: node %q does not reach root %q", start, t.root)
			}
			return nil
		}
		current = node.ParentID
	}
}

func (t *Tree) Root() Node {
	return t.nodes[t.root]
}

func (t *Tree) Node(id NodeID) (Node, bool) {
	node, ok := t.nodes[id]
	return node, ok
}

func (t *Tree) Children(parent NodeID) []Node {
	ids := t.children[parent]
	out := make([]Node, 0, len(ids))
	for _, id := range ids {
		out = append(out, t.nodes[id])
	}
	return out
}

func (t *Tree) Path(id NodeID) ([]Node, error) {
	if _, ok := t.nodes[id]; !ok {
		return nil, fmt.Errorf("hierarchy: unknown node %q", id)
	}

	var reverse []Node
	for current := id; current != ""; {
		node := t.nodes[current]
		reverse = append(reverse, node)
		current = node.ParentID
	}

	path := make([]Node, len(reverse))
	for i := range reverse {
		path[len(reverse)-1-i] = reverse[i]
	}
	return path, nil
}
