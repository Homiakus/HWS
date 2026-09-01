package domain

import "strings"

type NodeID string
type WorkspaceID string
type ResourceID string
type ActionID string

func validID[T ~string](id T) bool {
	return strings.TrimSpace(string(id)) != ""
}
