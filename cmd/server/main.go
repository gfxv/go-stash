package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gfxv/go-stash/internal/app"
	"github.com/gfxv/go-stash/pkg/cas"
)

func main() {
	storageOpts := cas.StorageOpts{
		BaseDir:  "stash",
		PathFunc: cas.DefaultTransformPathFunc,
		Pack:     cas.ZLibPack,
		Unpack:   cas.ZLibUnpack,
	}

	appOpts := &app.ApplicationOpts{
		Port:        5555,
		StorageOpts: storageOpts,
	}

	application := app.NewApp(appOpts)

	notifyReady := make(chan bool, 1)
	go func() {
		application.GRPC.MustRun(notifyReady)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop

	application.GRPC.Stop()
	log.Println("Gracefully stopped")
}
