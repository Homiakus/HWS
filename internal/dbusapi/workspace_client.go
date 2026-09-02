package dbusapi

import (
	"fmt"

	"github.com/Homiakus/HWS/internal/ipc"
)

func (c *Client) ActivateWorkspace(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("ActivateWorkspace", workspaceID, operationKey)
}

func (c *Client) RecoverWorkspace(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("RecoverWorkspace", workspaceID, operationKey)
}

func (c *Client) ResumeWorkspace(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("ResumeWorkspace", workspaceID, operationKey)
}

func (c *Client) SuspendWorkspace(workspaceID string) (string, error) {
	return c.stringArgCall("SuspendWorkspace", workspaceID)
}

func (c *Client) CloseWorkspace(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("CloseWorkspace", workspaceID, operationKey)
}

func (c *Client) WorkspaceStateJSON(workspaceID string) (string, error) {
	return c.stringArgCall("GetWorkspaceState", workspaceID)
}

func (c *Client) CompleteShellAction(payload string) error {
	if err := c.object.Call(ipc.InterfaceName+".CompleteShellAction", 0, payload).Err; err != nil {
		return fmt.Errorf("CompleteShellAction: %w", err)
	}
	return nil
}

func (c *Client) workspaceMutationCall(method, workspaceID, operationKey string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+"."+method, 0, workspaceID, operationKey).Store(&value); err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	return value, nil
}
