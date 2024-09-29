package sender

import (
	"bufio"
	"context"
	"fmt"
	"github.com/gfxv/go-stash/internal/services"
	"github.com/gfxv/go-stash/pkg/cas"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	gen "github.com/gfxv/go-stash/api"
	"github.com/gfxv/go-stash/pkg/dht"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const workerRatio = 3
const minWorkerCount = 1
const maxWorkerCount = 8

// for more info: https://github.com/grpc/grpc.github.io/issues/371
const fileChunkSize = 32 * 1024 // 32 KiB

type SenderOpts struct {
	Port          int
	SyncNode      string
	AnnounceNew   bool
	CheckInterval time.Duration

	Logger *slog.Logger

	NotifyRebase <-chan bool
}

type Client struct {
	opts   *SenderOpts
	logger *slog.Logger

	storageService *services.StorageService
	dhtService     *services.DHTService
}

func NewClient(
	opts *SenderOpts,
	storageService *services.StorageService,
	dhtService *services.DHTService,
) *Client {
	return &Client{
		opts:   opts,
		logger: opts.Logger,

		storageService: storageService,
		dhtService:     dhtService,
	}
}

func (c *Client) Serve(notifyReady chan<- bool) error {

	// TODO: break this into smaller parts

	if c.opts.SyncNode != "" {
		addr, err := net.ResolveTCPAddr("tcp", c.opts.SyncNode)
		if err != nil {
			return err
		}

		if err := c.LoadNodesFromSync(dht.NewNode(addr)); err != nil {
			return err
		}
	}

	if c.opts.AnnounceNew {
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", c.opts.Port))
		if err != nil {
			return err
		}
		if err := c.AnnounceNewNode(dht.NewNode(addr)); err != nil {
			c.logger.Error("error occurred while announcing new node", slog.Any("error", err.Error()))
			return err
		}
	}

	notifyReady <- true

	go func() {
		c.healthcheckLoop()
	}()

	go func() {
		for range c.opts.NotifyRebase {
			fmt.Println("rebase caught")
		}
	}()

	return nil
}

func (c *Client) AnnounceNewNode(node *dht.Node) error {
	if len(c.dhtService.GetNodes()) == 0 {
		return fmt.Errorf("cannot announce new node, Hash Ring is empty")
	}

	nodes := c.dhtService.GetNodes()
	for _, targetNode := range nodes {
		if err := c.newNodeRequest(node, targetNode); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) newNodeRequest(node, targetNode *dht.Node) error {
	conn, err := grpc.NewClient(targetNode.Addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gen.NewTransporterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	nodeInfo := &gen.NodeInfo{
		Address: node.Addr.String(),
		Alive:   false,
	}

	_, err = client.AnnounceNewNode(ctx, nodeInfo)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) handleRebaseSignal() error {
	offset := 0
	for {
		keys, err := c.storageService.GetKeysByChunks(offset)
		if err != nil {
			return err
		}

		rebaseInfo, err := c.checkForRebase(keys)
		if err != nil {
			return err
		}

		if err := c.copyStorage(rebaseInfo); err != nil {
			return err
		}

		if err = c.removeKeys(rebaseInfo); err != nil {
			return err
		}

		if len(keys) < cas.DB_CHUNK_SIZE {
			break
		}
		offset += cas.DB_CHUNK_SIZE
	}
	return nil
}

func (c *Client) copyStorage(rebaseInfo map[string]*dht.Node) error {
	for key, node := range rebaseInfo {
		if err := c.rebaseByKeyAndNode(key, node); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) checkForRebase(keys []string) (map[string]*dht.Node, error) {
	rebaseInfo := make(map[string]*dht.Node)
	selfAddr := fmt.Sprintf(":%d", c.opts.Port)

	for _, key := range keys {
		node, err := c.dhtService.GetNodeForKey(key)
		if err != nil {
			return nil, err
		}
		if !node.Alive {
			return nil, fmt.Errorf("node %v is not alive", node)
		}

		c.logger.Debug("Checking key for rebase",
			slog.String("key", key),
			slog.String("self address", selfAddr),
			slog.String("node address", node.Addr.String()),
		)

		if selfAddr == node.Addr.String() {
			continue
		}
		rebaseInfo[key] = node
	}

	return rebaseInfo, nil
}

func (c *Client) rebaseByKeyAndNode(key string, node *dht.Node) error {
	conn, err := grpc.NewClient(node.Addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gen.NewTransporterClient(conn)

	hashes, err := c.storageService.GetHashesByKey(key)
	if err != nil {
		return err
	}

	for _, hash := range hashes {
		if err := c.sendFile(client, key, hash); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) sendFile(
	client gen.TransporterClient,
	key, hash string,
) error {
	path := c.storageService.MakePathFromHash(hash)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// use WithTimeout + timeout depends on file size ?
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, err := client.SendChunks(ctx)
	if err != nil {
		return err
	}

	initRequest := makeSendChunksInitRequestBody(key, hash)
	if err = stream.Send(initRequest); err != nil {
		return err
	}

	if err = streamFileByChunks(file, stream); err != nil {
		return err
	}

	if _, err = stream.CloseAndRecv(); err != nil {
		return err
	}

	return nil
}

func streamFileByChunks(
	file *os.File,
	stream grpc.ClientStreamingClient[gen.Chunk, gen.StreamStatus],
) error {
	reader := bufio.NewReader(file)
	buffer := make([]byte, fileChunkSize)
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		chunk := &gen.Chunk{
			Data: &gen.Chunk_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		if err = stream.Send(chunk); err != nil {
			return err
		}
	}

	return nil
}

func makeSendChunksInitRequestBody(key, hash string) *gen.Chunk {
	return &gen.Chunk{
		Data: &gen.Chunk_Meta{
			Meta: &gen.Chunk_FileMetadata{
				Key:         key,
				ContentHash: &hash,
				FilePath:    nil,
				Compressed:  true,
			},
		},
	}
}

func (c *Client) removeKeys(info map[string]*dht.Node) error {
	for key := range info {
		if err := c.storageService.RemoveByKey(key); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) LoadNodesFromSync(syncNode *dht.Node) error {
	conn, err := grpc.NewClient(syncNode.Addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gen.NewTransporterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	stream, err := client.SyncNodes(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	nodes := make([]string, 0)
	for {
		nodeInfo, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		nodes = append(nodes, nodeInfo.Address)
	}

	return c.dhtService.LoadNodesFromList(nodes)
}

func (c *Client) healthcheckLoop() {
	for range time.Tick(c.opts.CheckInterval) {
		nodes := c.dhtService.GetNodes()
		if len(nodes) == 0 {
			continue
		}
		c.logger.Debug("starting HC dispatcher")
		_ = c.checkHealthDispatcher(nodes) // returns channel
		c.logger.Debug("done health checking")
	}
}

func (c *Client) checkHealthDispatcher(nodes map[int]*dht.Node) <-chan *dht.Node {
	jobs := make(chan *dht.Node, len(nodes))
	result := make(chan *dht.Node, len(nodes))
	for _, node := range nodes {
		jobs <- node
	}
	close(jobs)

	workerCount := calcWorkerCount(len(nodes))

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go c.checkHealthWorker(&wg, jobs, result)
	}

	wg.Wait()
	close(result)
	return result
}

func (c *Client) checkHealthWorker(wg *sync.WaitGroup, jobs <-chan *dht.Node, result chan<- *dht.Node) {
	defer wg.Done()

	for node := range jobs {
		err := c.makeHealthCheckRequest(node)
		node.Alive = err == nil
		result <- node
	}
}

func (c *Client) makeHealthCheckRequest(node *dht.Node) error {
	conn, err := grpc.NewClient(node.Addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	c.logger.Debug("requesting node health status", slog.String("address", node.Addr.String()))

	client := gen.NewHealthCheckerClient(conn)
	_, err = client.Healthcheck(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	return nil
}

func calcWorkerCount(nodesCount int) int {
	workerCount := nodesCount / workerRatio
	if workerCount > maxWorkerCount {
		return maxWorkerCount
	} else if workerCount < minWorkerCount {
		return minWorkerCount
	}
	return workerCount
}
