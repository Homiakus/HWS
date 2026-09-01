package contexttree

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
)

const SchemaVersion uint32 = 1

const DefaultSource = `{
  "schema": 1,
  "nodes": [
    {"id":"root","kind":"category","title":"Home","order":0},
    {"id":"dev","parentId":"root","kind":"category","title":"Development","order":10},
    {"id":"cad","parentId":"root","kind":"category","title":"CAD","order":20},
    {"id":"research","parentId":"root","kind":"category","title":"Research","order":30},
    {"id":"system","parentId":"root","kind":"category","title":"System","order":40},
    {"id":"personal","parentId":"root","kind":"category","title":"Personal","order":50},
    {"id":"dev-projects","parentId":"dev","kind":"category","title":"Projects","order":10},
    {"id":"dev-tools","parentId":"dev","kind":"category","title":"Tools","order":20},
    {"id":"cad-modeling","parentId":"cad","kind":"category","title":"Modeling","order":10},
    {"id":"cad-automation","parentId":"cad","kind":"category","title":"Automation","order":20},
    {"id":"research-literature","parentId":"research","kind":"category","title":"Literature","order":10},
    {"id":"research-experiments","parentId":"research","kind":"category","title":"Experiments","order":20},
    {"id":"system-network","parentId":"system","kind":"category","title":"Network","order":10},
    {"id":"system-services","parentId":"system","kind":"category","title":"Services","order":20},
    {"id":"system-logs","parentId":"system","kind":"category","title":"Logs","order":30},
    {"id":"personal-notes","parentId":"personal","kind":"category","title":"Notes","order":10},
    {"id":"personal-routines","parentId":"personal","kind":"category","title":"Routines","order":20}
  ]
}
`

type Node struct {
	ID          string `json:"id"`
	ParentID    string `json:"parentId,omitempty"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Order       int    `json:"order"`
	WorkspaceID string `json:"workspaceId,omitempty"`
}

type Document struct {
	Schema uint32 `json:"schema"`
	Nodes  []Node `json:"nodes"`
}

type Snapshot struct {
	Schema   uint32 `json:"schema"`
	Revision uint64 `json:"revision"`
	RootID   string `json:"rootId"`
	Nodes    []Node `json:"nodes"`
}

type Manager struct {
	mu sync.RWMutex

	path     string
	modTime  time.Time
	size     int64
	tree     *domain.Tree
	nodes    []Node
	revision uint64
	valid    bool
	lastErr  error
}

func (m *Manager) Configure(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("hierarchy config path is required")
	}
	if _, err := m.Apply([]byte(DefaultSource)); err != nil {
		return fmt.Errorf("compile built-in hierarchy: %w", err)
	}
	m.mu.Lock()
	m.path = path
	m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(DefaultSource), 0o600); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	_, err := m.Reload()
	return err
}

func (m *Manager) Apply(data []byte) (uint64, error) {
	tree, nodes, err := compile(data)
	m.mu.Lock()
	defer m.mu.Unlock()
	if err != nil {
		m.lastErr = err
		return m.revision, err
	}
	m.tree = tree
	m.nodes = nodes
	m.revision++
	m.valid = true
	m.lastErr = nil
	return m.revision, nil
}

func (m *Manager) Reload() (bool, error) {
	m.mu.RLock()
	path := m.path
	before := m.revision
	m.mu.RUnlock()
	if path == "" {
		return false, errors.New("hierarchy config path is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		return false, statErr
	}
	tree, nodes, compileErr := compile(data)

	m.mu.Lock()
	m.modTime = info.ModTime()
	m.size = info.Size()
	if compileErr != nil {
		m.lastErr = compileErr
		m.mu.Unlock()
		return false, compileErr
	}
	m.tree = tree
	m.nodes = nodes
	m.revision++
	m.valid = true
	m.lastErr = nil
	changed := m.revision != before
	m.mu.Unlock()
	return changed, nil
}

func (m *Manager) Poll() (bool, error) {
	m.mu.RLock()
	path, modTime, size := m.path, m.modTime, m.size
	m.mu.RUnlock()
	if path == "" {
		return false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Size() == size && info.ModTime().Equal(modTime) {
		return false, nil
	}
	return m.Reload()
}

func (m *Manager) Snapshot() (Snapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.valid || m.tree == nil {
		return Snapshot{}, false
	}
	return Snapshot{
		Schema:   SchemaVersion,
		Revision: m.revision,
		RootID:   string(m.tree.Root().ID),
		Nodes:    append([]Node(nil), m.nodes...),
	}, true
}

func (m *Manager) Path(id string) ([]Node, error) {
	m.mu.RLock()
	tree := m.tree
	valid := m.valid
	m.mu.RUnlock()
	if !valid || tree == nil {
		return nil, errors.New("hierarchy is unavailable")
	}
	path, err := tree.Path(domain.NodeID(id))
	if err != nil {
		return nil, err
	}
	out := make([]Node, 0, len(path))
	for _, item := range path {
		out = append(out, fromDomain(item))
	}
	return out, nil
}

func (m *Manager) LastError() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastErr
}

func (m *Manager) Revision() uint64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.revision
}

// Valid reports whether the current on-disk configuration is valid. A false
// value does not imply that Snapshot is unavailable: the last-known-good tree
// remains usable after a rejected reload.
func (m *Manager) Valid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.valid && m.lastErr == nil
}

func compile(data []byte) (*domain.Tree, []Node, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, fmt.Errorf("hierarchy config: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, nil, errors.New("hierarchy config: trailing JSON value")
		}
		return nil, nil, fmt.Errorf("hierarchy config: trailing data: %w", err)
	}
	if document.Schema != SchemaVersion {
		return nil, nil, fmt.Errorf("hierarchy config: unsupported schema %d", document.Schema)
	}
	if len(document.Nodes) == 0 || len(document.Nodes) > 4096 {
		return nil, nil, fmt.Errorf("hierarchy config: node count must be 1..4096")
	}

	domainNodes := make([]domain.Node, 0, len(document.Nodes))
	nodes := make([]Node, 0, len(document.Nodes))
	for _, raw := range document.Nodes {
		node, err := normalizeNode(raw)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
		domainNodes = append(domainNodes, domain.Node{
			ID:          domain.NodeID(node.ID),
			ParentID:    domain.NodeID(node.ParentID),
			Kind:        domain.NodeKind(node.Kind),
			Title:       node.Title,
			Order:       node.Order,
			WorkspaceID: domain.WorkspaceID(node.WorkspaceID),
		})
	}
	tree, err := domain.NewTree(domainNodes)
	if err != nil {
		return nil, nil, err
	}
	slices.SortFunc(nodes, func(a, b Node) int {
		if a.ParentID != b.ParentID {
			return strings.Compare(a.ParentID, b.ParentID)
		}
		if a.Order != b.Order {
			return a.Order - b.Order
		}
		if a.Title != b.Title {
			return strings.Compare(a.Title, b.Title)
		}
		return strings.Compare(a.ID, b.ID)
	})
	return tree, nodes, nil
}

func normalizeNode(node Node) (Node, error) {
	node.ID = strings.TrimSpace(node.ID)
	node.ParentID = strings.TrimSpace(node.ParentID)
	node.Kind = strings.TrimSpace(node.Kind)
	node.Title = strings.TrimSpace(node.Title)
	node.WorkspaceID = strings.TrimSpace(node.WorkspaceID)
	if node.Kind == "" {
		node.Kind = string(domain.NodeCategory)
	}
	if len(node.ID) == 0 || len(node.ID) > 128 {
		return Node{}, fmt.Errorf("hierarchy config: node id length must be 1..128")
	}
	if len(node.ParentID) > 128 || len(node.Title) == 0 || len(node.Title) > 256 || len(node.WorkspaceID) > 128 {
		return Node{}, fmt.Errorf("hierarchy config: invalid field length for node %q", node.ID)
	}
	switch domain.NodeKind(node.Kind) {
	case domain.NodeCategory, domain.NodeProject, domain.NodeTask, domain.NodeAction, domain.NodeWidget, domain.NodeQuery:
	default:
		return Node{}, fmt.Errorf("hierarchy config: node %q has invalid kind %q", node.ID, node.Kind)
	}
	return node, nil
}

func fromDomain(node domain.Node) Node {
	return Node{
		ID:          string(node.ID),
		ParentID:    string(node.ParentID),
		Kind:        string(node.Kind),
		Title:       node.Title,
		Order:       node.Order,
		WorkspaceID: string(node.WorkspaceID),
	}
}
