package ipc

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ProtocolVersion uint32 = 1
	BusName                = "org.homiakus.HWS1"
	ObjectPath             = "/org/homiakus/HWS1"
	InterfaceName          = "org.homiakus.HWS1"
)

type CapabilityLevel string

const (
	CapabilitySupported   CapabilityLevel = "supported"
	CapabilityConstrained CapabilityLevel = "constrained"
	CapabilityDegraded    CapabilityLevel = "degraded"
	CapabilityUnavailable CapabilityLevel = "unavailable"
)

type Capability struct {
	Level      CapabilityLevel `json:"level"`
	ReasonCode string          `json:"reasonCode,omitempty"`
}

type HelloRequest struct {
	ClientProtocol uint32 `json:"clientProtocol"`
	ClientInstance string `json:"clientInstance"`
}

type HelloResponse struct {
	ServerProtocol uint32                `json:"serverProtocol"`
	ServerInstance string                `json:"serverInstance"`
	RevisionEpoch  string                `json:"revisionEpoch"`
	Capabilities   map[string]Capability `json:"capabilities"`
}

func ValidateHello(request HelloRequest, response HelloResponse) error {
	if request.ClientProtocol != ProtocolVersion {
		return fmt.Errorf("protocol_incompatible: client protocol %d is not supported", request.ClientProtocol)
	}
	if response.ServerProtocol != ProtocolVersion {
		return fmt.Errorf("protocol_incompatible: server protocol %d is not supported", response.ServerProtocol)
	}
	if strings.TrimSpace(request.ClientInstance) == "" {
		return errors.New("invalid_client_instance")
	}
	if strings.TrimSpace(response.ServerInstance) == "" {
		return errors.New("invalid_server_instance")
	}
	if strings.TrimSpace(response.RevisionEpoch) == "" {
		return errors.New("invalid_revision_epoch")
	}
	for name, capability := range response.Capabilities {
		if strings.TrimSpace(name) == "" {
			return errors.New("invalid_capability_name")
		}
		switch capability.Level {
		case CapabilitySupported, CapabilityConstrained, CapabilityDegraded, CapabilityUnavailable:
		default:
			return fmt.Errorf("invalid_capability_level: %s", capability.Level)
		}
	}
	return nil
}

type CacheIdentity struct {
	ServerInstance string
	RevisionEpoch  string
}

func (c CacheIdentity) ValidFor(response HelloResponse) bool {
	return c.ServerInstance != "" &&
		c.RevisionEpoch != "" &&
		c.ServerInstance == response.ServerInstance &&
		c.RevisionEpoch == response.RevisionEpoch
}

type MutationRequest struct {
	WorkspaceID        string `json:"workspaceId"`
	DefinitionRevision string `json:"definitionRevision"`
	OperationKey       string `json:"operationKey"`
}

func (r MutationRequest) Validate() error {
	if strings.TrimSpace(r.WorkspaceID) == "" {
		return errors.New("workspace_id_required")
	}
	if strings.TrimSpace(r.DefinitionRevision) == "" {
		return errors.New("definition_revision_required")
	}
	if strings.TrimSpace(r.OperationKey) == "" {
		return errors.New("operation_key_required")
	}
	return nil
}

type MutationAck struct {
	OperationID string `json:"operationId"`
}

func (a MutationAck) Validate() error {
	if strings.TrimSpace(a.OperationID) == "" {
		return errors.New("operation_id_required")
	}
	return nil
}

type ErrorCode string

const (
	ErrProtocolIncompatible      ErrorCode = "protocol_incompatible"
	ErrStaleRevision             ErrorCode = "stale_revision"
	ErrWorkspaceNotFound         ErrorCode = "workspace_not_found"
	ErrWorkspaceDefinition       ErrorCode = "workspace_definition_invalid"
	ErrCapabilityUnavailable     ErrorCode = "capability_unavailable"
	ErrOperationConflict         ErrorCode = "operation_conflict"
	ErrDaemonRecovering          ErrorCode = "daemon_recovering"
	ErrInternal                  ErrorCode = "internal_error"
)
