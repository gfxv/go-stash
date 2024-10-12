package services

import (
	"github.com/gfxv/go-stash/pkg/dht"
	"net"
)

// DHTService struct encapsulates a hash ring, which is responsible for managing
// the distribution of nodes within the DHT.
type DHTService struct {
	ring *dht.HashRing
}

// NewDHTService creates a new instance of DHTService.
func NewDHTService(ring *dht.HashRing) *DHTService {
	return &DHTService{ring: ring}
}

// GetNodesAddr retrieves the addresses of all nodes in the DHT ring.
//
// This method iterates through the nodes managed by the DHT's hash ring and
// collects their addresses into a slice of strings.
func (s *DHTService) GetNodesAddr() []string {
	nodes := make([]string, 0)
	for _, node := range s.ring.GetNodes() {
		nodes = append(nodes, node.Addr.String())
	}
	return nodes
}

// LoadNodesFromList adds a list of nodes to the DHT ring from a slice of address strings.
//
// This method takes a slice of node addresses as strings, resolves each address
// to a TCP address, and adds the corresponding nodes to the DHT hash ring.
// If any address fails to resolve, the method returns an error immediately,
// and no nodes are added to the ring for that entry.
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

// AddNode adds a single node to the DHT ring.
//
// This method takes a pointer to a `dht.Node` and adds it to the DHT hash ring.
// The addition is safe for concurrent modifications, allowing multiple goroutines
// to add nodes without causing race conditions.
func (s *DHTService) AddNode(node *dht.Node) {
	s.ring.AddNode(node)
}

// RemoveNode removes a specified node from the DHT ring.
//
// This method takes a pointer to a `dht.Node` and removes it from the DHT hash ring.
// The removal is safe for concurrent modifications, allowing multiple goroutines
// to remove nodes without causing race conditions.
func (s *DHTService) RemoveNode(node *dht.Node) {
	s.ring.RemoveNode(node)
}

// GetNodeForKey retrieves the node responsible for a given key in the DHT ring.
//
// This method takes a string key as input and returns the node that is responsible
// for that key according to the DHT hash ring. If no node can be found for the
// provided key (probably because of hash ring is empty), the method will return an error.
func (s *DHTService) GetNodeForKey(key string) (*dht.Node, error) {
	return s.ring.GetNodeForKey(key)
}

// NodeExists checks if a node responsible for a given key exists in the DHT ring.
//
// This method takes a string key as input and returns a boolean indicating whether
// a node exists for that key in the DHT hash ring. It is a quick way to verify
// the presence of a node without retrieving it.
func (s *DHTService) NodeExists(key string) bool {
	return s.ring.NodeExists(key)
}

// GetNodes retrieves all nodes in the DHT ring.
//
// This method returns a map of nodes currently present in the DHT hash ring.
// The keys in the map are integers representing node identifiers, and the values
// are pointers to `dht.Node` instances.
func (s *DHTService) GetNodes() map[int]*dht.Node {
	return s.ring.GetNodes()
}
