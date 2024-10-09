package app

import (
	"fmt"
	grpcapp "github.com/gfxv/go-stash/internal/app/grpc"
	senderapp "github.com/gfxv/go-stash/internal/app/sender"
	"github.com/gfxv/go-stash/internal/config"
	"github.com/gfxv/go-stash/internal/sender"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/internal/utils"
	"github.com/gfxv/go-stash/pkg/cas"
	"github.com/gfxv/go-stash/pkg/dht"
	"log/slog"
	"net"
)

type ApplicationOpts struct {
	GRPCOpts    config.GRPCConfig
	StorageOpts cas.StorageOpts
}

type App struct {
	GRPC   *grpcapp.App
	Sender *senderapp.App
}

func NewApp(logger *slog.Logger, opts *ApplicationOpts) *App {
	storage, err := cas.NewDefaultStorage(opts.StorageOpts)
	if err != nil {
		panic(err)
	}

	ring, errs := loadRingFromConfig(&opts.GRPCOpts)
	if len(errs) != 0 {
		utils.HandleFatal(logger, "can't load nodes from config", errs...)
	}

	notifyRebase := make(chan bool)
	replicationChan := make(chan *cas.KeyHashPair)

	storageService := services.NewStorageService(storage)
	dhtService := services.NewDHTService(ring)

	senderOpts := sender.SenderOpts{
		Port:              opts.GRPCOpts.Port,
		CheckInterval:     opts.GRPCOpts.HealthCheckInterval,
		SyncNode:          opts.GRPCOpts.SyncNode,
		AnnounceNew:       opts.GRPCOpts.AnnounceNewNode,
		ReplicationFactor: opts.StorageOpts.ReplicationFactor,
		Logger:            logger,
		NotifyRebase:      notifyRebase,
		ReplicationChan:   replicationChan,
	}
	senderApp := senderapp.New(&senderOpts, storageService, dhtService)
	grpcOpts := grpcapp.GRPCOpts{
		Port:         opts.GRPCOpts.Port,
		Logger:       logger,
		NotifyRebase: notifyRebase,
	}
	grpcApp := grpcapp.New(&grpcOpts, storageService, dhtService)

	return &App{
		GRPC:   grpcApp,
		Sender: senderApp,
	}
}

func loadRingFromConfig(cfg *config.GRPCConfig) (*dht.HashRing, []error) {
	nodes := cfg.Nodes
	ring := dht.NewHashRing()
	errorNodes := make([]error, 0)

	for _, n := range nodes {
		addr, err := net.ResolveTCPAddr("tcp", n)
		if err != nil {
			errorNodes = append(errorNodes, fmt.Errorf("error resolving node address: %s", err.Error()))
		}
		ring.AddNode(dht.NewNode(addr))
	}
	return ring, errorNodes
}
