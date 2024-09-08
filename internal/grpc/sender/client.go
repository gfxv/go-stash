package sender

import (
	"context"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"

	gen "github.com/gfxv/go-stash/api"
	"github.com/gfxv/go-stash/pkg/dht"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const workerRatio = 3
const minWorkerCount = 1
const maxWorkerCount = 8

type Client struct {
}

func (c *Client) CheckHealthDispatcher(nodes []*dht.Node) <-chan *dht.Node {
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
	// workerCoount = max(workerCoount, minCount)
	// workerCoount = min(workerCoount, maxCount)
	// ??????????

	workerCount := nodesCount / workerRatio
	if workerCount > maxWorkerCount {
		return maxWorkerCount
	} else if workerCount < minWorkerCount {
		return minWorkerCount
	}
	return workerCount
}
