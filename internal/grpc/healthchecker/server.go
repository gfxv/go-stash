package healthchecker

import (
	"context"
	gen "github.com/gfxv/go-stash/api"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type serverAPI struct {
	gen.UnimplementedHealthCheckerServer
}

func Register(gRPC *grpc.Server) {
	gen.RegisterHealthCheckerServer(gRPC, &serverAPI{})
}

func (s *serverAPI) Healthcheck(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// just a healthcheck of a node
	// no need for something special
	return &emptypb.Empty{}, nil
}
