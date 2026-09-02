package dbusapi

import (
	"fmt"

	"github.com/Homiakus/HWS/internal/ipc"
)

func (c *Client) ActivateWorkspace(workspaceID, operationKey string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+".ActivateWorkspace", 0, workspaceID, operationKey).Store(&value); err != nil {
		return "", fmt.Errorf("ActivateWorkspace: %w", err)
	}
	return value, nil
}

func (c *Client) WorkspaceStateJSON(workspaceID string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+".GetWorkspaceState", 0, workspaceID).Store(&value); err != nil {
		return "", fmt.Errorf("GetWorkspaceState: %w", err)
	}
	return value, nil
}

func (c *Client) CompleteShellAction(payload string) error {
	if err := c.object.Call(ipc.InterfaceName+".CompleteShellAction", 0, payload).Err; err != nil {
		return fmt.Errorf("CompleteShellAction: %w", err)
	}
	return nil
}
