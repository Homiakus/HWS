package dbusapi

import (
	"encoding/json"
	"fmt"

	"github.com/Homiakus/HWS/internal/ipc"
	"github.com/godbus/dbus/v5"
)

type Client struct {
	conn          *dbus.Conn
	object        dbus.BusObject
	instance      string
	server        string
	revisionEpoch string
	capabilities  map[string]string
}

func ConnectSession() (*Client, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect HWS session bus: %w", err)
	}
	client := &Client{
		conn:         conn,
		object:       conn.Object(ipc.BusName, dbus.ObjectPath(ipc.ObjectPath)),
		instance:     randomID(),
		capabilities: map[string]string{},
	}
	var protocol uint32
	var capabilities string
	if err := client.object.Call(ipc.InterfaceName+".Hello", 0, ipc.ProtocolVersion, client.instance).Store(
		&protocol,
		&client.server,
		&client.revisionEpoch,
		&capabilities,
	); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("HWS Hello: %w", err)
	}
	if protocol != ipc.ProtocolVersion {
		_ = conn.Close()
		return nil, fmt.Errorf("HWS protocol mismatch: server=%d client=%d", protocol, ipc.ProtocolVersion)
	}
	if capabilities != "" {
		if err := json.Unmarshal([]byte(capabilities), &client.capabilities); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("decode HWS capabilities: %w", err)
		}
	}
	return client, nil
}

func (c *Client) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *Client) ServerInstance() string { return c.server }
func (c *Client) RevisionEpoch() string  { return c.revisionEpoch }

func (c *Client) Capabilities() map[string]string {
	out := make(map[string]string, len(c.capabilities))
	for key, value := range c.capabilities {
		out[key] = value
	}
	return out
}

func (c *Client) HealthJSON() (string, error) {
	return c.stringCall("GetHealth")
}

func (c *Client) PanelJSON() (string, error) {
	return c.stringCall("GetPanelSnapshot")
}

func (c *Client) SpecJSON() (string, error) {
	return c.stringCall("GetPanelSpec")
}

func (c *Client) TreeJSON() (string, error) {
	return c.stringCall("GetTree")
}

func (c *Client) PathJSON(nodeID string) (string, error) {
	return c.stringArgCall("GetPath", nodeID)
}

func (c *Client) ApplicationJSON(appID string) (string, error) {
	return c.stringArgCall("GetApplicationSurface", appID)
}

func (c *Client) WorkspaceStateJSON(workspaceID string) (string, error) {
	return c.stringArgCall("GetWorkspaceState", workspaceID)
}

func (c *Client) ActivateWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("ActivateWorkspace", workspaceID, operationKey)
}

func (c *Client) RecoverWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("RecoverWorkspace", workspaceID, operationKey)
}

func (c *Client) ResumeWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("ResumeWorkspace", workspaceID, operationKey)
}

func (c *Client) SuspendWorkspaceJSON(workspaceID string) (string, error) {
	return c.stringArgCall("SuspendWorkspace", workspaceID)
}

func (c *Client) CloseWorkspaceJSON(workspaceID, operationKey string) (string, error) {
	return c.workspaceMutationCall("CloseWorkspace", workspaceID, operationKey)
}

func (c *Client) ReloadPanel() (bool, string, error) {
	var ok bool
	var diagnostic string
	if err := c.object.Call(ipc.InterfaceName+".ReloadPanel", 0).Store(&ok, &diagnostic); err != nil {
		return false, "", fmt.Errorf("ReloadPanel: %w", err)
	}
	return ok, diagnostic, nil
}

func (c *Client) workspaceMutationCall(method, workspaceID, operationKey string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+"."+method, 0, workspaceID, operationKey).Store(&value); err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	return value, nil
}

func (c *Client) stringArgCall(method, arg string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+"."+method, 0, arg).Store(&value); err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	return value, nil
}

func (c *Client) stringCall(method string) (string, error) {
	var value string
	if err := c.object.Call(ipc.InterfaceName+"."+method, 0).Store(&value); err != nil {
		return "", fmt.Errorf("%s: %w", method, err)
	}
	return value, nil
}
