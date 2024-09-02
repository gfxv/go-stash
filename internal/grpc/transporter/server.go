package transporter

import (
	"bytes"
	"context"
	gen "github.com/gfxv/go-stash/api"
	"github.com/gfxv/go-stash/pkg/cas"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
)

type serverAPI struct {
	gen.UnimplementedTransporterServer
	storage *cas.Storage
}

func Register(gRPC *grpc.Server, storage *cas.Storage) {
	gen.RegisterTransporterServer(gRPC, &serverAPI{storage: storage})
}

func (s *serverAPI) SendChunks(stream gen.Transporter_SendChunksServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "can't receive key: %v", err)
	}
	key := req.GetMeta().GetKey()                 // key
	contentHash := req.GetMeta().GetContentHash() // hash = filepath

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

	// save data on disk
	// NOTE: data already has a header and been compressed
	contentPath := s.storage.MakePathFromHash(contentHash)
	err = s.storage.PrepareParentFolders(contentPath)
	if err != nil {
		return status.Errorf(codes.Internal, "can't prepare parent folders: %v", err)
	}
	err = s.storage.Write(contentPath, buffer.Bytes())
	if err != nil {
		return status.Errorf(codes.Internal, "can't store file file to storage: %v", err)
	}

	// save path to meta.db
	err = s.storage.AddNewPath(key, contentHash)
	if err != nil {
		return status.Errorf(codes.Internal, "can't store key-hash pair")
	}

	// TODO: ...
	return stream.SendAndClose(&gen.StreamStatus{
		Size: 0,
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
