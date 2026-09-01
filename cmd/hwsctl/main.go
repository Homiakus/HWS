package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"

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
	case "app":
		if len(args) != 1 {
			return errors.New("usage: hwsctl app <application-id>")
		}
		return withClient(func(client *dbusapi.Client) error {
			value, err := client.ApplicationJSON(args[0])
			return printJSON(value, err)
		})
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
	Status         string `json:"status"`
	ConfigValid    bool   `json:"configValid"`
	ConfigError    string `json:"configError"`
	Applications   int    `json:"applications"`
	Windows        int    `json:"windows"`
	Views          int    `json:"views"`
	PanelRevision  uint64 `json:"panelRevision"`
	SurfaceRevision string `json:"surfaceRevision"`
	Providers      []struct {
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
		if !health.ConfigValid {
			return errors.New("doctor found an invalid panel configuration")
		}
		return nil
	})
}

func printUsage() {
	fmt.Print(`hwsctl commands:
  demo                 run the deterministic headless workspace demo (default)
  health               print daemon health JSON
  doctor               print a human-readable daemon/provider diagnostic
  panel                print the current panel snapshot
  spec                 print the normalized Panel DSL spec
  app <application-id> print one aggregated ApplicationSurface
  reload               reload Panel DSL configuration
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
