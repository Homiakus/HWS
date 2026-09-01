package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Homiakus/HWS/internal/daemon"
	"github.com/Homiakus/HWS/internal/dbusapi"
	"github.com/Homiakus/HWS/internal/providers"
	"github.com/Homiakus/HWS/internal/providers/launcherentry"
	"github.com/Homiakus/HWS/internal/providers/mpris"
	providerserver "github.com/Homiakus/HWS/internal/providers/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		return errors.New("XDG_RUNTIME_DIR is required; refusing insecure /tmp provider socket fallback")
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configHome = filepath.Join(home, ".config")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	registry := providers.NewRegistry()
	hub := daemon.NewHub(registry)
	providerSocket := filepath.Join(runtimeDir, "hws", "providers.sock")
	providerServer := providerserver.New(providerSocket, hub)
	hub.SetActions(providerServer.Actions)

	panelPath := filepath.Join(configHome, "hws", "panel.hws.hcl")
	if err := hub.Configure(panelPath); err != nil {
		log.Printf("panel config rejected; using last-known-good: %v", err)
	}

	dbusServer, err := dbusapi.OpenSession(hub)
	if err != nil {
		return err
	}
	defer dbusServer.Close()
	hub.SetNotifiers(dbusServer.EmitPanelChanged, dbusServer.EmitPanelConfigChanged)

	mprisCollector, err := mpris.OpenSession(hub)
	if err != nil {
		log.Printf("MPRIS collector unavailable: %v", err)
	} else {
		defer mprisCollector.Close()
		go mprisCollector.Run(ctx, 2*time.Second, func(err error) {
			log.Printf("MPRIS collector: %v", err)
		})
	}

	launcherCollector, err := launcherentry.OpenSession(hub)
	if err != nil {
		log.Printf("LauncherEntry collector unavailable: %v", err)
	} else {
		defer launcherCollector.Close()
		go launcherCollector.Run(ctx, func(err error) {
			log.Printf("LauncherEntry collector: %v", err)
		})
	}

	errs := make(chan error, 1)
	go func() {
		if err := providerServer.Serve(ctx); err != nil {
			select {
			case errs <- fmt.Errorf("provider server: %w", err):
			default:
			}
		}
	}()
	go hub.RunMaintenance(ctx, time.Second, func(err error) {
		log.Printf("maintenance: %v", err)
	})

	log.Printf("hwsd ready: bus=%s provider_socket=%s panel=%s", "org.homiakus.HWS1", providerSocket, panelPath)
	select {
	case <-ctx.Done():
		return nil
	case err := <-errs:
		return err
	}
}
