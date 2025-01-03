package transporter

import (
	"bytes"
	"context"
	"fmt"
	gen "github.com/gfxv/go-stash/api"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/pkg/cas"
	"github.com/gfxv/go-stash/pkg/dht"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"net"
)

type serverAPI struct {
	gen.UnimplementedTransporterServer
	storageService *services.StorageService
	dhtService     *services.DHTService

	notifyRebase    chan<- bool
	replicationChan chan<- *cas.KeyHashPair // TODO: add replicationChan to app configuration !!!
}

func Register(
	gRPC *grpc.Server,
	storageService *services.StorageService,
	dhtService *services.DHTService,
	notifyRebase chan<- bool,
	replicationChan chan<- *cas.KeyHashPair,
) {
	gen.RegisterTransporterServer(gRPC, &serverAPI{
		storageService:  storageService,
		dhtService:      dhtService,
		notifyRebase:    notifyRebase,
		replicationChan: replicationChan,
	})
}

func (s *serverAPI) GetDestination(
	ctx context.Context,
	keyRequest *gen.KeyRequest,
) (*gen.NodeInfo, error) {
	key := keyRequest.GetKey()
	if len(key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "key is empty")
	}

	node, err := s.dhtService.GetNodeForKey(key)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if !node.Alive {
		return nil, status.Error(codes.Internal, fmt.Sprintf("node corresponding for key '%s' is unavailable", key))
	}

	return &gen.NodeInfo{
		Address: node.Addr.String(),
		Alive:   node.Alive,
	}, nil
}

// SendChunks receives a stream of file chunks (or whole file)
// from client and stores it on disk
func (s *serverAPI) SendChunks(stream gen.Transporter_SendChunksServer) error {
	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "can't receive key: %v", err)
	}

	key := req.GetMeta().GetKey()
	if len(key) == 0 {
		return status.Errorf(codes.InvalidArgument, "empty key")
	}
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
		if chunk == nil || len(chunk) == 0 {
			return status.Errorf(codes.InvalidArgument, "empty chunk")
		}
		_, err = buffer.Write(chunk)
		if err != nil {
			return status.Errorf(codes.Internal, "can't write chunk data: %v", err)
		}
	}

	var contentHash string
	if compressed {
		contentHash = req.GetMeta().GetContentHash()
		if len(contentHash) == 0 {
			return status.Errorf(codes.InvalidArgument, "empty hash")
		}

		err := s.storageService.SaveCompressed(key, contentHash, buffer.Bytes())
		if err != nil {
			return status.Errorf(codes.Internal, "can't save compressed file: %v", err)
		}
	} else {
		path := req.GetMeta().GetFilePath()
		if len(path) == 0 {
			return status.Errorf(codes.InvalidArgument, "empty path")
		}
		file := &cas.File{
			Path: path,
			Data: buffer.Bytes(),
		}
		contentHash, err = s.storageService.SaveRaw(key, file)
		if err != nil {
			return status.Errorf(codes.Internal, "can't save raw file: %v", err)
		}
	}

	fmt.Println("replicate:", req.GetMeta().GetReplicate())
	if req.GetMeta().GetReplicate() {
		go func() { // костыль?
			fmt.Println("sending signal to replication channel")
			keyHashPair := &cas.KeyHashPair{Key: key, Hash: contentHash}
			s.replicationChan <- keyHashPair
		}()
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
	if len(key) == 0 {
		return nil, status.Error(codes.InvalidArgument, "empty key")
	}

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
	hash := chunkRequest.GetHash()
	if len(hash) == 0 {
		return status.Error(codes.InvalidArgument, "empty hash")
	}
	needDecompression := chunkRequest.GetNeedDecompression()

	fileContent, err := s.storageService.GetFileDataByHash(hash, needDecompression)
	if err != nil {
		return status.Errorf(codes.NotFound, "can't get file with hash %s: %v", hash, err)
	}

	// TODO: add byte splitting and streaming file in chunks
	chunk := &gen.ReceiveChunkResponse{Data: fileContent}
	if err := stream.Send(chunk); err != nil {
		return status.Errorf(codes.Unknown, "can't send chunk: %v", err)
	}

	return nil
}

// SyncNodes ...
func (s *serverAPI) SyncNodes(_ *emptypb.Empty, stream gen.Transporter_SyncNodesServer) error {
	nodes := s.dhtService.GetNodes()
	for _, node := range nodes {
		nodeInfo := &gen.NodeInfo{
			Address: node.Addr.String(),
			Alive:   node.Alive,
		}
		if err := stream.Send(nodeInfo); err != nil {
			return err
		}
	}
	return nil
}

// Rebase ...
func (s *serverAPI) Rebase(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	// anything other than that ???
	s.notifyRebase <- true
	return nil, nil
}

// AnnounceNewNode ...
func (s *serverAPI) AnnounceNewNode(
	ctx context.Context,
	newNode *gen.NodeInfo,
) (*emptypb.Empty, error) {

	// check if node exists
	if s.dhtService.NodeExists(newNode.GetAddress()) { // node already exists in hash ring
		return nil, status.Errorf(codes.AlreadyExists, "node already exists in DHT")
	}

	// add node to the hash ring
	addr, err := net.ResolveTCPAddr("tcp", newNode.GetAddress())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "can't resolve address: %v", err)
	}

	s.dhtService.AddNode(dht.NewNode(addr)) // return err ?

	return nil, nil
}

// AnnounceRemoveNode ...
func (s *serverAPI) AnnounceRemoveNode(
	ctx context.Context,
	deadNode *gen.NodeInfo,
) (*emptypb.Empty, error) {

	if !s.dhtService.NodeExists(deadNode.GetAddress()) {
		return nil, status.Errorf(codes.NotFound, "node does not exist in DHT")
	}

	addr, err := net.ResolveTCPAddr("tcp", deadNode.GetAddress())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "can't resolve address: %v", err)
	}

	s.dhtService.RemoveNode(dht.NewNode(addr)) // return err ?

	return nil, nil
}
