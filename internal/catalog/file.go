package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Homiakus/HWS/internal/domain"
)

const WorkspaceSchemaVersion uint32 = 1

const DefaultWorkspaceSource = `{
  "schema": 1,
  "active": {},
  "workspaces": []
}
`

type workspaceResourceDocument struct {
	ID               string              `json:"id"`
	Kind             domain.ResourceKind `json:"kind"`
	Required         bool                `json:"required"`
	Ownership        domain.Ownership    `json:"ownership"`
	DesktopAppID     string              `json:"desktopAppId,omitempty"`
	Executable       string              `json:"executable,omitempty"`
	Args             []string            `json:"args,omitempty"`
	WorkingDirectory string              `json:"workingDirectory,omitempty"`
	Placement        *placementDocument  `json:"placement,omitempty"`
}

type placementDocument struct {
	MonitorRole string       `json:"monitorRole"`
	Workspace   int          `json:"workspace"`
	Rect        rectDocument `json:"rect"`
}

type rectDocument struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type workspaceDocument struct {
	ID        string                      `json:"id"`
	Revision  string                      `json:"revision"`
	Resources []workspaceResourceDocument `json:"resources"`
}

type workspaceFileDocument struct {
	Schema     uint32              `json:"schema"`
	Active     map[string]string   `json:"active"`
	Workspaces []workspaceDocument `json:"workspaces"`
}

type WorkspaceSnapshot struct {
	Schema          uint32            `json:"schema"`
	Revision        uint64            `json:"revision"`
	Active          map[string]string `json:"active"`
	DefinitionCount int               `json:"definitionCount"`
}

type File struct {
	mu sync.RWMutex

	path        string
	modTime     time.Time
	size        int64
	revision    uint64
	valid       bool
	lastErr     error
	definitions map[string]domain.DesiredState
	active      map[domain.WorkspaceID]string
}

func NewFile() *File {
	return &File{}
}

func definitionKey(id domain.WorkspaceID, revision string) string {
	return string(id) + "@" + revision
}

func (f *File) Configure(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("workspace catalog path is required")
	}
	if _, err := f.Apply([]byte(DefaultWorkspaceSource)); err != nil {
		return fmt.Errorf("compile built-in workspace catalog: %w", err)
	}
	f.mu.Lock()
	f.path = path
	f.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(DefaultWorkspaceSource), 0o600); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	_, err := f.Reload()
	return err
}

func (f *File) Apply(data []byte) (uint64, error) {
	definitions, active, err := compileWorkspaceFile(data)
	f.mu.Lock()
	defer f.mu.Unlock()
	if err != nil {
		f.lastErr = err
		return f.revision, err
	}
	f.definitions = definitions
	f.active = active
	f.revision++
	f.valid = true
	f.lastErr = nil
	return f.revision, nil
}

func (f *File) Reload() (bool, error) {
	f.mu.RLock()
	path := f.path
	before := f.revision
	f.mu.RUnlock()
	if path == "" {
		return false, errors.New("workspace catalog path is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	definitions, active, compileErr := compileWorkspaceFile(data)

	f.mu.Lock()
	f.modTime = info.ModTime()
	f.size = info.Size()
	if compileErr != nil {
		f.lastErr = compileErr
		f.mu.Unlock()
		return false, compileErr
	}
	f.definitions = definitions
	f.active = active
	f.revision++
	f.valid = true
	f.lastErr = nil
	changed := f.revision != before
	f.mu.Unlock()
	return changed, nil
}

func (f *File) Poll() (bool, error) {
	f.mu.RLock()
	path, modTime, size := f.path, f.modTime, f.size
	f.mu.RUnlock()
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
	return f.Reload()
}

func (f *File) RunMaintenance(ctx context.Context, interval time.Duration, report func(error)) {
	if interval <= 0 {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := f.Poll(); err != nil && report != nil {
				report(err)
			}
		}
	}
}

func (f *File) Resolve(id domain.WorkspaceID, revision string) (domain.DesiredState, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	desired, ok := f.definitions[definitionKey(id, revision)]
	if !ok {
		return domain.DesiredState{}, fmt.Errorf("catalog: workspace %s revision %s not found", id, revision)
	}
	return cloneDesired(desired), nil
}

func (f *File) Current(id domain.WorkspaceID) (domain.DesiredState, error) {
	f.mu.RLock()
	revision, ok := f.active[id]
	if !ok {
		f.mu.RUnlock()
		return domain.DesiredState{}, fmt.Errorf("catalog: workspace %s has no active revision", id)
	}
	desired, ok := f.definitions[definitionKey(id, revision)]
	f.mu.RUnlock()
	if !ok {
		return domain.DesiredState{}, fmt.Errorf("catalog: active workspace %s revision %s is unavailable", id, revision)
	}
	return cloneDesired(desired), nil
}

func (f *File) Snapshot() WorkspaceSnapshot {
	f.mu.RLock()
	defer f.mu.RUnlock()
	active := make(map[string]string, len(f.active))
	for id, revision := range f.active {
		active[string(id)] = revision
	}
	return WorkspaceSnapshot{
		Schema:          WorkspaceSchemaVersion,
		Revision:        f.revision,
		Active:          active,
		DefinitionCount: len(f.definitions),
	}
}

func (f *File) Revision() uint64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.revision
}

func (f *File) Valid() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.valid && f.lastErr == nil
}

func (f *File) LastError() error {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.lastErr
}

func compileWorkspaceFile(data []byte) (map[string]domain.DesiredState, map[domain.WorkspaceID]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var document workspaceFileDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, nil, fmt.Errorf("workspace catalog: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, nil, errors.New("workspace catalog: trailing JSON value")
		}
		return nil, nil, fmt.Errorf("workspace catalog: trailing data: %w", err)
	}
	if document.Schema != WorkspaceSchemaVersion {
		return nil, nil, fmt.Errorf("workspace catalog: unsupported schema %d", document.Schema)
	}
	if len(document.Workspaces) > 4096 || len(document.Active) > 2048 {
		return nil, nil, errors.New("workspace catalog: size limit exceeded")
	}

	definitions := make(map[string]domain.DesiredState, len(document.Workspaces))
	for _, raw := range document.Workspaces {
		id := domain.WorkspaceID(strings.TrimSpace(raw.ID))
		revision := strings.TrimSpace(raw.Revision)
		resources := make([]domain.ResourceSpec, 0, len(raw.Resources))
		for _, resource := range raw.Resources {
			spec := domain.ResourceSpec{
				ID:               domain.ResourceID(strings.TrimSpace(resource.ID)),
				Kind:             resource.Kind,
				Required:         resource.Required,
				Ownership:        resource.Ownership,
				DesktopAppID:     strings.TrimSpace(resource.DesktopAppID),
				Executable:       strings.TrimSpace(resource.Executable),
				Args:             append([]string(nil), resource.Args...),
				WorkingDirectory: strings.TrimSpace(resource.WorkingDirectory),
			}
			if resource.Placement != nil {
				spec.Placement = &domain.PlacementIntent{
					MonitorRole: strings.TrimSpace(resource.Placement.MonitorRole),
					Workspace:   resource.Placement.Workspace,
					Rect: domain.NormalizedRect{
						X:      resource.Placement.Rect.X,
						Y:      resource.Placement.Rect.Y,
						Width:  resource.Placement.Rect.Width,
						Height: resource.Placement.Rect.Height,
					},
				}
			}
			resources = append(resources, spec)
		}
		desired := domain.DesiredState{WorkspaceID: id, Revision: revision, Resources: resources}
		if err := desired.Validate(); err != nil {
			return nil, nil, fmt.Errorf("workspace catalog: %w", err)
		}
		key := definitionKey(id, revision)
		if _, exists := definitions[key]; exists {
			return nil, nil, fmt.Errorf("workspace catalog: duplicate definition %s", key)
		}
		definitions[key] = desired
	}

	active := make(map[domain.WorkspaceID]string, len(document.Active))
	for rawID, rawRevision := range document.Active {
		id := domain.WorkspaceID(strings.TrimSpace(rawID))
		revision := strings.TrimSpace(rawRevision)
		if id == "" || revision == "" {
			return nil, nil, errors.New("workspace catalog: active workspace id and revision are required")
		}
		if _, ok := definitions[definitionKey(id, revision)]; !ok {
			return nil, nil, fmt.Errorf("workspace catalog: active workspace %s references missing revision %s", id, revision)
		}
		active[id] = revision
	}
	return definitions, active, nil
}

func cloneDesired(in domain.DesiredState) domain.DesiredState {
	out := in
	out.Resources = make([]domain.ResourceSpec, len(in.Resources))
	for i := range in.Resources {
		out.Resources[i] = in.Resources[i]
		out.Resources[i].Args = append([]string(nil), in.Resources[i].Args...)
		if in.Resources[i].Placement != nil {
			placement := *in.Resources[i].Placement
			out.Resources[i].Placement = &placement
		}
	}
	return out
}
