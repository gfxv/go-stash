package app

import (
	"fmt"
	grpcapp "github.com/gfxv/go-stash/internal/app/grpc"
	"github.com/gfxv/go-stash/pkg/cas"
)

type ApplicationOpts struct {
	Port        int
	StorageRoot string
}

type App struct {
	GRPC *grpcapp.App
}

func NewApp(opts *ApplicationOpts) *App {
	storage, err := cas.NewDefaultStorage(opts.StorageRoot)
	if err != nil {
		fmt.Println("app.app")
		panic(err)
	}

	grpcApp := grpcapp.New(opts.Port, storage)

	return &App{
		GRPC: grpcApp,
	}
}
