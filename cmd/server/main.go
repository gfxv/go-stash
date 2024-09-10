package main

import (
	"fmt"
	"github.com/gfxv/go-stash/internal/config"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gfxv/go-stash/internal/app"
	"github.com/gfxv/go-stash/pkg/cas"
)

func main() {

	cfg := config.MustLoad()
	fmt.Println(cfg)

	storageOpts := cas.StorageOpts{
		BaseDir:  cfg.Storage.Path,
		PathFunc: cas.DefaultTransformPathFunc,
		Pack:     cas.ZLibPack,
		Unpack:   cas.ZLibUnpack,
	}

	appOpts := &app.ApplicationOpts{
		Port:        cfg.GRPC.Port,
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
