package grpcapp

import (
	"fmt"
	"github.com/gfxv/go-stash/internal/grpc/transporter"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/pkg/cas"
	"google.golang.org/grpc"
	"log"
	"net"
)

type App struct {
	grpcServer *grpc.Server
	port       int

	storage cas.Storage
}

// New creates new gRPC server app
func New(port int, storage *services.StorageService) *App {
	server := grpc.NewServer()
	transporter.Register(server, storage)

	return &App{
		port:       port,
		grpcServer: server,
	}
}

// MustRun runs gRPC server and panics if any error occurs
func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
	}
}

// Run runs gRPC server
func (a *App) Run() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Println("grpc server listening on", l.Addr())

	if err := a.grpcServer.Serve(l); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return err
}

// Stop stops gRPC server
func (a *App) Stop() {
	a.grpcServer.GracefulStop()
}
