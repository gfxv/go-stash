package sender

import (
	"context"
	"fmt"
	"github.com/gfxv/go-stash/internal/services"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"net"
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

type SenderOpts struct {
	SyncNode      string
	CheckInterval time.Duration
}

type Client struct {
	opts       *SenderOpts
	dhtService *services.DHTService
}

func NewClient(opts *SenderOpts, dhtService *services.DHTService) *Client {
	return &Client{opts: opts, dhtService: dhtService}
}

func (c *Client) Serve(notifyReady chan<- bool) error {
	if c.opts.SyncNode != "" {
		addr, err := net.ResolveTCPAddr("tcp", c.opts.SyncNode)
		if err != nil {
			return err
		}

		if err := c.LoadNodesFromSync(dht.NewNode(addr)); err != nil {
			return err
		}
	}

	// mb run in goroutine ?

	notifyReady <- true

	c.healthcheckLoop()

	return nil
}

func (c *Client) healthcheckLoop() {
	for range time.Tick(c.opts.CheckInterval) {
		nodes := c.dhtService.GetNodes()
		if len(nodes) == 0 {
			continue
		}
		_ = c.CheckHealthDispatcher(nodes) // returns channel
	}
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

	fmt.Println("making request")
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

func (c *Client) CheckHealthDispatcher(nodes map[int]*dht.Node) <-chan *dht.Node {
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
		err := makeHealthCheckRequest(node)
		node.Alive = err == nil
		result <- node
	}
}

func makeHealthCheckRequest(node *dht.Node) error {
	conn, err := grpc.NewClient(node.Addr.String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gen.NewHealthCheckerClient(conn)
	_, err = client.Healthcheck(context.Background(), &emptypb.Empty{})
	if err != nil {
		return err
	}
	return nil
}

func calcWorkerCount(nodesCount int) int {
	// or make like:
	// workerCount = max(workerCount, minCount)
	// workerCount = min(workerCount, maxCount)
	// ??????????

	workerCount := nodesCount / workerRatio
	if workerCount > maxWorkerCount {
		return maxWorkerCount
	} else if workerCount < minWorkerCount {
		return minWorkerCount
	}
	return workerCount
}
