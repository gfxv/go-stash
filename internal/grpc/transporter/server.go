package transporter

import (
	"bytes"
	"context"
	gen "github.com/gfxv/go-stash/api"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/pkg/cas"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
)

type serverAPI struct {
	gen.UnimplementedTransporterServer
	storageService *services.StorageService
}

func Register(gRPC *grpc.Server, storageService *services.StorageService) {
	gen.RegisterTransporterServer(gRPC, &serverAPI{storageService: storageService})
}

func (s *serverAPI) SendChunks(stream gen.Transporter_SendChunksServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "can't receive key: %v", err)
	}

	key := req.GetMeta().GetKey() // key
	compressed := req.GetMeta().GetCompressed()

	buffer := bytes.Buffer{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Unknown, "can't receive chunk: %v", err)
		}

		chunk := req.GetChunkData()
		_, err = buffer.Write(chunk)
		if err != nil {
			return status.Errorf(codes.Internal, "can't write chunk data: %v", err)
		}
	}

	if compressed {
		contentHash := req.GetMeta().GetContentHash()
		err := s.storageService.SaveCompressed(key, contentHash, buffer.Bytes())
		if err != nil {
			return status.Errorf(codes.Internal, "can't save compressed file: %v", err)
		}
	} else {
		file := &cas.File{
			Path: req.GetMeta().GetFilePath(),
			Data: buffer.Bytes(),
		}
		err := s.storageService.SaveRaw(key, file)
		if err != nil {
			return status.Errorf(codes.Internal, "can't save raw file: %v", err)
		}
	}

	return stream.SendAndClose(&gen.StreamStatus{
		Size: uint32(len(buffer.Bytes())),
	})
}

func (s *serverAPI) SyncNodes(_ *emptypb.Empty, stream gen.Transporter_SyncNodesServer) error {
	// read nodes from file or dht
	// stream response
	return nil
}

func (s *serverAPI) AnnounceNewNode(
	ctx context.Context,
	newNode *gen.NodeInfo,
) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *serverAPI) AnnounceRemoveNode(
	ctx context.Context,
	deadNode *gen.NodeInfo,
) (*emptypb.Empty, error) {
	return nil, nil
}
