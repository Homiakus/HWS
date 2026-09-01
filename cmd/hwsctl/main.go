package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Homiakus/HWS/internal/adapters/fake"
	"github.com/Homiakus/HWS/internal/application/reconcile"
	"github.com/Homiakus/HWS/internal/catalog"
	"github.com/Homiakus/HWS/internal/domain"
	workspaceflow "github.com/Homiakus/HWS/internal/orchestration/workspace"
)

func main() {
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
		log.Fatal(err)
	}
	desktop := fake.NewDesktop()
	lifecycle, err := workspaceflow.Open(definitions, reconcile.New(desktop))
	if err != nil {
		log.Fatal(err)
	}

	if err := lifecycle.Activate(ctx, desired.WorkspaceID, desired.Revision, "activate:local-dev:v1:demo"); err != nil {
		log.Fatal(err)
	}
	state, err := lifecycle.State(ctx, desired.WorkspaceID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("workspace=%s status=%s required=%d/%d failure=%s\n",
		state.WorkspaceID,
		state.Status,
		state.ReachedRequired,
		state.TotalRequired,
		state.LastFailureCode,
	)
}
