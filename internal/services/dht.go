package services

import (
	"github.com/gfxv/go-stash/pkg/dht"
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

func (s *DHTService) AddNode(node *dht.Node) {
	s.ring.AddNode(node)
}

func (s *DHTService) RemoveNode(node *dht.Node) {
	s.ring.RemoveNode(node)
}

func (s *DHTService) GetNodeForKey(key string) (*dht.Node, error) {
	return s.ring.GetNodeForKey(key)
}

func (s *DHTService) NodeExists(key string) bool {
	return s.ring.NodeExists(key)
}

func (s *DHTService) GetNodes() map[int]*dht.Node {
	return s.ring.GetNodes()
}
