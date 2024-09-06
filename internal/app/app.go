package app

import (
	grpcapp "github.com/gfxv/go-stash/internal/app/grpc"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/pkg/cas"
)

type ApplicationOpts struct {
	Port        int
	StorageOpts cas.StorageOpts
}

type App struct {
	GRPC *grpcapp.App
}

func NewApp(opts *ApplicationOpts) *App {
	storage, err := cas.NewDefaultStorage(opts.StorageOpts)
	if err != nil {
		panic(err)
	}
	storageService := services.NewStorageService(storage)
	grpcApp := grpcapp.New(opts.Port, storageService)

	return &App{
		GRPC: grpcApp,
	}
}
