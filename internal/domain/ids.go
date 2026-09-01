package domain

import "strings"

type NodeID string
type WorkspaceID string
type ResourceID string
type ActionID string
type SurfaceID string
type SurfaceWindowID string
type SurfaceViewID string
type SurfaceProviderID string

func validID[T ~string](id T) bool {
	return strings.TrimSpace(string(id)) != ""
}
