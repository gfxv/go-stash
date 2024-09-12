package grpcapp

import (
	"fmt"
	"log"
	"net"

	"github.com/gfxv/go-stash/internal/grpc/healthchecker"
	"github.com/gfxv/go-stash/internal/grpc/transporter"
	"github.com/gfxv/go-stash/internal/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type App struct {
	grpcServer *grpc.Server
	port       int
}

// New creates new gRPC server app
func New(port int, storage *services.StorageService, dht *services.DHTService) *App {
	server := grpc.NewServer()
	healthchecker.Register(server)
	transporter.Register(server, storage, dht)

	reflection.Register(server)

	return &App{
		port:       port,
		grpcServer: server,
	}
}

// MustRun runs gRPC server and panics if any error occurs
func (a *App) MustRun(notifyReady chan<- bool) {
	if err := a.Run(notifyReady); err != nil {
		panic(err)
	}
}

// Run runs gRPC server
func (a *App) Run(notifyReady chan<- bool) error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Println("grpc server listening on", l.Addr())

	notifyReady <- true
	if err := a.grpcServer.Serve(l); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return err
}

// Stop stops gRPC server
func (a *App) Stop() {
	a.grpcServer.GracefulStop()
}
