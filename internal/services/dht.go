package services

import (
	"github.com/gfxv/go-stash/pkg/dht"
	"log/slog"
	"net"
)

type DHTService struct {
	ring *dht.HashRing
}

func NewDHTService(ring *dht.HashRing) *DHTService {
	return &DHTService{ring: ring}
}

func (s *DHTService) GetNodesAddr() []string {
	nodes := make([]string, 0)
	for _, node := range s.ring.GetNodes() {
		nodes = append(nodes, node.Addr.String())
	}
	return nodes
}

func (s *DHTService) LoadNodesFromList(nodes []string) error {
	for _, node := range nodes {
		addr, err := net.ResolveTCPAddr("tcp", node)
		if err != nil {
			return err
		}
		// AddNode safe for concurrent modification
		s.ring.AddNode(dht.NewNode(addr))
	}
	return nil
}

func (s *DHTService) GetNodes() map[int]*dht.Node {
	return s.ring.GetNodes()
}

func (s *DHTService) Status(logger *slog.Logger) {
	// TODO: ?
}
