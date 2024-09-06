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

// SendChunks receives a stream of file chunks (or whole file)
// from client and stores it on disk
func (s *serverAPI) SendChunks(stream gen.Transporter_SendChunksServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "can't receive key: %v", err)
	}

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

	key := req.GetMeta().GetKey()
	compressed := req.GetMeta().GetCompressed()

	// TODO: check if optional fields exist (!!!)

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

// ReceiveInfo returns hashes that have same key
func (s *serverAPI) ReceiveInfo(
	ctx context.Context,
	infoRequest *gen.ReceiveInfoRequest,
) (*gen.ReceiveInfoResponse, error) {

	key := infoRequest.GetKey()

	hashes, err := s.storageService.GetHashesByKey(key)
	if err != nil {
		// mb codes.Internal is better ...?
		return nil, status.Errorf(codes.NotFound, "can't get files: %v", err)
	}

	response := &gen.ReceiveInfoResponse{
		Size:   uint32(len(hashes)),
		Hashes: hashes,
	}

	return response, nil
}

// ReceiveChunks sends a stream of file chunks
// (or whole file) based on provided hash
func (s *serverAPI) ReceiveChunks(
	chunkRequest *gen.ReceiveChunkRequest,
	stream gen.Transporter_ReceiveChunksServer,
) error {
	return nil
}

// SyncNodes ...
func (s *serverAPI) SyncNodes(_ *emptypb.Empty, stream gen.Transporter_SyncNodesServer) error {
	// read nodes from file or dht
	// stream response
	return nil
}

// AnnounceNewNode ...
func (s *serverAPI) AnnounceNewNode(
	ctx context.Context,
	newNode *gen.NodeInfo,
) (*emptypb.Empty, error) {
	return nil, nil
}

// AnnounceRemoveNode ...
func (s *serverAPI) AnnounceRemoveNode(
	ctx context.Context,
	deadNode *gen.NodeInfo,
) (*emptypb.Empty, error) {
	return nil, nil
}
