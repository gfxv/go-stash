package main

import (
	"github.com/gfxv/go-stash/internal/config"
	"github.com/gfxv/go-stash/pkg/slogger"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gfxv/go-stash/internal/app"
	"github.com/gfxv/go-stash/pkg/cas"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func main() {
	// Load application configuration
	cfg := config.MustLoad()

	// Set up logging
	logger := setupLogger(cfg.Env)
	cfg.Validate(logger)

	// Prepare options
	storageOpts := cas.StorageOpts{
		BaseDir:           cfg.Storage.Path,
		PathFunc:          cas.DefaultTransformPathFunc,
		Pack:              cas.ZLibPack,
		Unpack:            cas.ZLibUnpack,
		ReplicationFactor: cfg.Storage.ReplicationFactor,
	}
	appOpts := &app.ApplicationOpts{
		GRPCOpts:    cfg.GRPC,
		StorageOpts: storageOpts,
	}

	application := app.NewApp(logger, appOpts)

	// Channel to notify when the GRPC server is ready
	notifyReady := make(chan bool, 2)
	go func() {
		application.GRPC.MustRun(notifyReady)
	}()

	// Wait for the GRPC server to be ready
	<-notifyReady

	go func() {
		// Run the sender service
		application.Sender.MustRun(notifyReady)
	}()

	// Wait for the sender service to be ready
	<-notifyReady

	// Channel to listen for OS termination signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a termination signal
	<-stop

	// Stop the GRPC server gracefully
	application.GRPC.Stop()
	log.Println("Gracefully stopped")
}

func setupLogger(env string) *slog.Logger {
	var l *slog.Logger

	switch env {
	case envDev:
		l = slog.New(slogger.NewSloggerHandler(
			os.Stdout, slogger.SloggerHandlerOpts{
				SlogOpts: &slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelDebug,
				},
			}))
	case envProd:
		l = slog.New(slogger.NewSloggerHandler(
			os.Stdout, slogger.SloggerHandlerOpts{
				SlogOpts: &slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelInfo,
				},
			}))
	}
	return l
}
