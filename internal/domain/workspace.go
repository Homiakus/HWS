package domain

import (
	"errors"
	"fmt"
	"strings"
)

type ResourceKind string

const (
	ResourceDesktopApp ResourceKind = "desktop_app"
	ResourceProcess    ResourceKind = "process"
	ResourceTerminal   ResourceKind = "terminal"
)

type Ownership string

const (
	OwnershipManaged  Ownership = "managed"
	OwnershipAdopted  Ownership = "adopted"
	OwnershipExternal Ownership = "external"
)

type NormalizedRect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (r NormalizedRect) Valid() bool {
	return r.X >= 0 && r.Y >= 0 && r.Width > 0 && r.Height > 0 && r.X+r.Width <= 1 && r.Y+r.Height <= 1
}

type LogicalRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func (r LogicalRect) Valid() bool {
	return r.Width > 0 && r.Height > 0
}

type PlacementIntent struct {
	MonitorRole string
	Workspace   int
	Rect        NormalizedRect
}

type ResolvedPlacement struct {
	TopologyRevision string      `json:"topologyRevision"`
	MonitorRef       string      `json:"monitorRef"`
	MonitorIndex     int         `json:"monitorIndex"`
	Workspace        int         `json:"workspace"`
	Rect             LogicalRect `json:"rect"`
}

type PlacementObservation struct {
	TopologyRevision string      `json:"topologyRevision,omitempty"`
	MonitorRef       string      `json:"monitorRef,omitempty"`
	Workspace        int         `json:"workspace,omitempty"`
	Rect             LogicalRect `json:"rect"`
	Reached          bool        `json:"reached"`
}

type ResourceSpec struct {
	ID               ResourceID
	Kind             ResourceKind
	Required         bool
	Ownership        Ownership
	DesktopAppID     string
	Executable       string
	Args             []string
	WorkingDirectory string
	Placement        *PlacementIntent
}

type DesiredState struct {
	WorkspaceID WorkspaceID
	Revision    string
	Resources   []ResourceSpec
}

func (d DesiredState) Validate() error {
	if !validID(d.WorkspaceID) {
		return errors.New("workspace: workspace id is required")
	}
	if strings.TrimSpace(d.Revision) == "" {
		return errors.New("workspace: revision is required")
	}
	seen := make(map[ResourceID]struct{}, len(d.Resources))
	for _, resource := range d.Resources {
		if !validID(resource.ID) {
			return errors.New("workspace: resource id is required")
		}
		if _, exists := seen[resource.ID]; exists {
			return fmt.Errorf("workspace: duplicate resource id %q", resource.ID)
		}
		seen[resource.ID] = struct{}{}
		switch resource.Ownership {
		case OwnershipManaged, OwnershipAdopted, OwnershipExternal:
		default:
			return fmt.Errorf("workspace: resource %q has invalid ownership %q", resource.ID, resource.Ownership)
		}
		switch resource.Kind {
		case ResourceDesktopApp:
			if strings.TrimSpace(resource.DesktopAppID) == "" {
				return fmt.Errorf("workspace: desktop app %q requires desktop app id", resource.ID)
			}
		case ResourceProcess, ResourceTerminal:
			if strings.TrimSpace(resource.Executable) == "" {
				return fmt.Errorf("workspace: process resource %q requires executable", resource.ID)
			}
		default:
			return fmt.Errorf("workspace: resource %q has unsupported kind %q", resource.ID, resource.Kind)
		}
		if resource.Placement != nil {
			if !resource.Placement.Rect.Valid() {
				return fmt.Errorf("workspace: resource %q has invalid normalized placement", resource.ID)
			}
			if resource.Placement.Workspace < 0 {
				return fmt.Errorf("workspace: resource %q has invalid workspace index", resource.ID)
			}
		}
	}
	return nil
}

type ResourceObservation struct {
	ResourceID ResourceID
	Present    bool
	Ready      bool
	Ownership  Ownership
	SessionRef string
	AppID      string
	ReasonCode string
	Placement  *PlacementObservation
}

type ObservedState struct {
	WorkspaceID      WorkspaceID
	TopologyRevision string
	Resources        map[ResourceID]ResourceObservation
}
