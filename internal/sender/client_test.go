package sender

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/gfxv/go-stash/internal/app"
	"github.com/gfxv/go-stash/pkg/cas"
	"github.com/gfxv/go-stash/pkg/dht"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	ports := []int{5555, 5556, 5557}

	node1, err := makeNode(ports[0])
	assert.NoError(t, err, "error in creating node")
	node2, err := makeNode(ports[1])
	assert.NoError(t, err, "error in creating node")
	node3, err := makeNode(ports[2])
	assert.NoError(t, err, "error in creating node")

	stop1 := make(chan bool, 1)
	stop2 := make(chan bool, 1)
	stop3 := make(chan bool, 1)
	notifyRunning := make(chan bool, len(ports))

	go runServer(ports[0], stop1, notifyRunning)
	go runServer(ports[1], stop2, notifyRunning)
	go runServer(ports[2], stop3, notifyRunning)

	// waiting until all nodes are up and running
	for i := 0; i < len(ports); i++ {
		<-notifyRunning
	}

	ring := dht.NewHashRing()
	ring.AddNode(node1, node2, node3)

	c := Client{}
	nodeStatus := c.CheckHealthDispatcher(ring.GetNodes())
	for node := range nodeStatus {
		// all nodes should be alive
		assert.True(t, node.Alive)
	}

	// stopping 2nd server (node)
	stop2 <- true

	nodeStatus = c.CheckHealthDispatcher(ring.GetNodes())
	for node := range nodeStatus {
		// check if 2nd node is down
		if node.Addr.String() == fmt.Sprintf(":%d", ports[1]) {
			assert.False(t, node.Alive)
			continue
		}
		// check other nodes
		assert.True(t, node.Alive)
	}
}

func runServer(port int, stop chan bool, notifyRunning chan<- bool) {
	storageOpts := cas.StorageOpts{
		BaseDir:  fmt.Sprintf("test/stash-%d", port),
		PathFunc: cas.DefaultTransformPathFunc,
		Pack:     cas.ZLibPack,
		Unpack:   cas.ZLibUnpack,
	}

	// TODO: remove later
	// (move creating to NewDefaultStorage)
	if err := os.Mkdir(storageOpts.BaseDir, os.ModePerm); err != nil {
		panic(err)
	}

	appOpts := &app.ApplicationOpts{
		Port:        port,
		StorageOpts: storageOpts,
	}
	application := app.NewApp(appOpts)

	go func() {
		application.GRPC.MustRun(notifyRunning)
	}()

	<-stop
	application.GRPC.Stop()
}

func makeNode(port int) (*dht.Node, error) {
	a, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	return dht.NewNode(a), nil
}
