package main

import (
	"github.com/gfxv/go-stash/internal/app"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	appOpts := &app.ApplicationOpts{
		Port:        5555,
		StorageRoot: "stash",
	}

	application := app.NewApp(appOpts)

	go func() {
		application.GRPC.MustRun()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	application.GRPC.Stop()
	log.Println("Gracefully stopped")
}
