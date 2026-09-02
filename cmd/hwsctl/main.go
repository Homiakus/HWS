package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Homiakus/HWS/internal/adapters/fake"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
	"github.com/Homiakus/HWS/internal/dbusapi"
	"github.com/Homiakus/HWS/internal/domain"
	workspaceflow "github.com/Homiakus/HWS/internal/orchestration/workspace"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	command := "demo"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "demo":
		return runDemo()
	case "health":
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.HealthJSON()
			return printJSON(value, err)
		})
	case "panel":
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.PanelJSON()
			return printJSON(value, err)
		})
	case "spec":
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.SpecJSON()
			return printJSON(value, err)
		})
	case "tree":
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.TreeJSON()
			return printJSON(value, err)
		})
	case "path":
		if len(args) != 1 {
			return errors.New("usage: hwsctl path <node-id>")
		}
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.PathJSON(args[0])
			return printJSON(value, err)
		})
	case "app":
		if len(args) != 1 {
			return errors.New("usage: hwsctl app <application-id>")
		}
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.ApplicationJSON(args[0])
			return printJSON(value, err)
		})
	case "workspace":
		return runWorkspace(args)
	case "reload":
		return withClient(func(client *dbusapi.Client) error {
			ok, diagnostic, err := client.ReloadPanel()
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("panel config rejected: %s", diagnostic)
			}
			fmt.Println("panel configuration reloaded")
			return nil
		})
	case "doctor":
		return runDoctor()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q; run hwsctl help", command)
	}
}

func runWorkspace(args []string) error {
	if len(args) < 2 {
		return errors.New("usage: hwsctl workspace <activate|state|recover|resume|suspend|close> <workspace-id> [operation-key]")
	}
	action := args[0]
	workspaceID := strings.TrimSpace(args[1])
	if workspaceID == "" {
		return errors.New("workspace id is required")
	}
	return withClient(func(client *dbusapi.Client) error {
		switch action {
		case "state":
			if len(args) != 2 {
				return errors.New("usage: hwsctl workspace state <workspace-id>")
			}
			value, err := client.WorkspaceStateJSON(workspaceID)
			return printJSON(value, err)
		case "suspend":
			if len(args) != 2 {
				return errors.New("usage: hwsctl workspace suspend <workspace-id>")
			}
			value, err := client.SuspendWorkspace(workspaceID)
			return printJSON(value, err)
		case "activate", "recover", "resume", "close":
			if len(args) > 3 {
				return fmt.Errorf("usage: hwsctl workspace %s <workspace-id> [operation-key]", action)
			}
			operationKey := operationKey(action, workspaceID, args[2:])
			var value string
			var err error
			switch action {
			case "activate":
				value, err = client.ActivateWorkspace(workspaceID, operationKey)
			case "recover":
				value, err = client.RecoverWorkspace(workspaceID, operationKey)
			case "resume":
				value, err = client.ResumeWorkspace(workspaceID, operationKey)
			case "close":
				value, err = client.CloseWorkspace(workspaceID, operationKey)
			}
			if err == nil {
				fmt.Fprintf(os.Stderr, "operation-key: %s\n", operationKey)
			}
			return printJSON(value, err)
		default:
			return fmt.Errorf("unknown workspace action %q", action)
		}
	})
}

func operationKey(action, workspaceID string, optional []string) string {
	if len(optional) == 1 && strings.TrimSpace(optional[0]) != "" {
		return strings.TrimSpace(optional[0])
	}
	return fmt.Sprintf("%s:%s:%d", action, workspaceID, time.Now().UTC().UnixNano())
}

func withClient(run func(*dbusapi.Client) error) error {
	client, err := dbusapi.ConnectSession()
	if err != nil {
		return err
	}
	defer client.Close()
	return run(client)
}

func printJSON(value string, err error) error {
	if err != nil {
		return err
	}
	var payload any
	if json.Unmarshal([]byte(value), &payload) != nil {
		fmt.Println(value)
		return nil
	}
	formatted, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(formatted))
	return nil
}

type healthReport struct {
	Status                   string `json:"status"`
	ConfigValid              bool   `json:"configValid"`
	ConfigError              string `json:"configError"`
	HierarchyValid           bool   `json:"hierarchyValid"`
	HierarchyError           string `json:"hierarchyError"`
	HierarchyRevision        uint64 `json:"hierarchyRevision"`
	WorkspaceCatalogValid    bool   `json:"workspaceCatalogValid"`
	WorkspaceCatalogError    string `json:"workspaceCatalogError"`
	WorkspaceCatalogRevision uint64 `json:"workspaceCatalogRevision"`
	WorkspaceDefinitions     int    `json:"workspaceDefinitions"`
	WorkspaceLifecycleReady  bool   `json:"workspaceLifecycleReady"`
	Applications             int    `json:"applications"`
	Windows                  int    `json:"windows"`
	Views                    int    `json:"views"`
	PanelRevision            uint64 `json:"panelRevision"`
	SurfaceRevision          string `json:"surfaceRevision"`
	Providers                []struct {
		ProviderID string `json:"providerId"`
		Kind       string `json:"kind"`
		Connected  bool   `json:"connected"`
		Stale      bool   `json:"stale"`
		Partial    bool   `json:"partial"`
		Revision   uint64 `json:"revision"`
	} `json:"providers"`
}

func runDoctor() error {
	return withClient(func(client *dbusapi.Client) error {
		raw, err := client.HealthJSON()
		if err != nil {
			return err
		}
		var health healthReport
		if err := json.Unmarshal([]byte(raw), &health); err != nil {
			return fmt.Errorf("decode daemon health: %w", err)
		}
		fmt.Printf("HWS daemon: %s\n", health.Status)
		fmt.Printf("server: %s  epoch: %s\n", client.ServerInstance(), client.RevisionEpoch())
		fmt.Printf("surface: apps=%d windows=%d views=%d panel-revision=%d\n", health.Applications, health.Windows, health.Views, health.PanelRevision)
		if health.ConfigValid {
			fmt.Println("panel config: valid")
		} else {
			fmt.Printf("panel config: INVALID (%s)\n", health.ConfigError)
		}
		if health.HierarchyValid {
			fmt.Printf("hierarchy: valid revision=%d\n", health.HierarchyRevision)
		} else {
			fmt.Printf("hierarchy: INVALID (%s)\n", health.HierarchyError)
		}
		if health.WorkspaceCatalogValid {
			fmt.Printf("workspaces: valid revision=%d definitions=%d lifecycle-ready=%t\n", health.WorkspaceCatalogRevision, health.WorkspaceDefinitions, health.WorkspaceLifecycleReady)
		} else {
			fmt.Printf("workspaces: INVALID (%s)\n", health.WorkspaceCatalogError)
		}
		sort.Slice(health.Providers, func(i, j int) bool {
			return health.Providers[i].ProviderID < health.Providers[j].ProviderID
		})
		for _, provider := range health.Providers {
			state := "connected"
			switch {
			case provider.Stale:
				state = "stale"
			case provider.Partial:
				state = "partial"
			case !provider.Connected:
				state = "disconnected"
			}
			fmt.Printf("provider %-22s %-10s kind=%s rev=%d\n", provider.ProviderID, state, provider.Kind, provider.Revision)
		}
		if !health.ConfigValid || !health.HierarchyValid || !health.WorkspaceCatalogValid || !health.WorkspaceLifecycleReady {
			return errors.New("doctor found an invalid or unavailable subsystem")
		}
		return nil
	})
}

func printUsage() {
	fmt.Print(`hwsctl commands:
  demo                                      run the deterministic headless workspace demo (default)
  health                                    print daemon health JSON
  doctor                                    print a human-readable daemon/provider diagnostic
  panel                                     print the current panel snapshot
  spec                                      print the normalized Panel DSL spec
  tree                                      print the context hierarchy snapshot
  path <node-id>                            print a hierarchy path from root to node
  app <application-id>                      print one aggregated ApplicationSurface
  workspace state <id>                      print durable workspace lifecycle state
  workspace activate <id> [operation-key]   activate the catalog's current revision
  workspace recover <id> [operation-key]    reconcile a degraded/failed workspace
  workspace resume <id> [operation-key]     resume/reconcile a suspended workspace
  workspace suspend <id>                    mark workspace inactive without closing resources (v1 semantics)
  workspace close <id> [operation-key]      close HWS-managed resources only
  reload                                    reload Panel DSL configuration

Supplying an operation key makes retries explicitly idempotent. If omitted, hwsctl generates a unique key.
`)
}

func runDemo() error {
	ctx := context.Background()
	desired := domain.DesiredState{
		WorkspaceID: "local-dev",
		Revision:    "v1",
		Resources: []domain.ResourceSpec{
			{
				ID:           "editor",
				Kind:         domain.ResourceDesktopApp,
				Required:     true,
				Ownership:    domain.OwnershipManaged,
				DesktopAppID: "dev.zed.Zed.desktop",
			},
			{
				ID:               "terminal",
				Kind:             domain.ResourceTerminal,
				Required:         true,
				Ownership:        domain.OwnershipManaged,
				Executable:       "bash",
				WorkingDirectory: ".",
			},
		},
	}

	definitions := catalog.NewMemory()
	if err := definitions.Put(desired); err != nil {
		return err
	}
	desktop := fake.NewDesktop()
	lifecycle, err := workspaceflow.Open(definitions, reconcile.New(desktop))
	if err != nil {
		return err
	}

	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:local-dev:v1:demo"); err != nil {
		return err
	}
	state, err := lifecycle.State(ctx, desired.WorkspaceID)
	if err != nil {
		return err
	}

	fmt.Printf("workspace=%s status=%s required=%d/%d failure=%s\n",
		state.WorkspaceID,
		state.Status,
		state.ReachedRequired,
		state.TotalRequired,
		state.LastFailureCode,
	)
	return nil
}
