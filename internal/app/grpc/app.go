package grpcapp

import (
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"log/slog"
	"net"

	"github.com/gfxv/go-stash/internal/grpc/healthchecker"
	"github.com/gfxv/go-stash/internal/grpc/transporter"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCOpts struct {
	Port   int
	Logger *slog.Logger

	NotifyRebase chan<- bool
}

type App struct {
	grpcServer *grpc.Server
	port       int
}

// New creates new gRPC server app
func New(opts *GRPCOpts, storage *services.StorageService, dht *services.DHTService) *App {

	logOpts := []logging.Option{
		logging.WithLogOnEvents(
			logging.StartCall, logging.FinishCall,
			logging.PayloadReceived, logging.PayloadSent,
		),
	}

	recoveryOpts := []recovery.Option{
		recovery.WithRecoveryHandler(func(p any) error {
			opts.Logger.Error("recovered from panic", slog.Any("panic", p))
			return status.Errorf(codes.Internal, "internal error")
		}),
	}

	server := grpc.NewServer(grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(recoveryOpts...),
		logging.UnaryServerInterceptor(InterceptorLogger(opts.Logger), logOpts...),
	))

	healthchecker.Register(server)
	transporter.Register(server, storage, dht, opts.NotifyRebase)

	reflection.Register(server)

	return &App{
		port:       opts.Port,
		grpcServer: server,
	}
}

// InterceptorLogger adapts slog logger to interceptor logger.
func InterceptorLogger(l *slog.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		l.Log(ctx, slog.Level(lvl), msg, fields...)
	})
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
