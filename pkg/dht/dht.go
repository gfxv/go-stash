package dht

import (
	"fmt"
	"hash/fnv"
	"net"
	"sync"
)

type Node struct {
	Addr  net.Addr
	Alive bool
}

func NewNode(addr net.Addr) *Node {
	return &Node{addr, false}
}

func HashKey(key string) int {
	h := fnv.New64()
	_, err := h.Write([]byte(key))
	if err != nil {
		panic("stash: can't get hash of nodes address") // TODO: remove panic
	}
	return int(h.Sum64())
}

type HashRing struct {
	mu    sync.Mutex
	ids   []int
	nodes map[int]*Node
}

// NewHashRing creates empty HashRing
func NewHashRing() *HashRing {
	return &HashRing{
		mu:    sync.Mutex{},
		ids:   make([]int, 0),
		nodes: make(map[int]*Node),
	}
}

// AddNode adds node to Hash Ring
func (h *HashRing) AddNode(nodes ...*Node) {
	for _, node := range nodes {
		nodeKey := HashKey(node.Addr.String())
		h.mu.Lock()
		h.insertId(nodeKey)
		h.nodes[nodeKey] = node
		h.mu.Unlock()
	}
}

// GetNodeForKey returns Node corresponding to given key.
// Error can occur if Node is not found
func (h *HashRing) GetNodeForKey(key string) (*Node, error) {
	hashedKey := HashKey(key)
	closestNodeId := findClosestElement(h.ids, hashedKey)
	node, ok := h.nodes[closestNodeId]
	if !ok {
		return nil, fmt.Errorf("stash: DHT Node for key '%s' not found", key)
	}
	return node, nil
}

// NodeExists returns true if node exists in hash ring, false otherwise
func (h *HashRing) NodeExists(key string) bool {
	hashedKey := HashKey(key)
	_, ok := h.nodes[hashedKey]
	return ok
}

func (h *HashRing) GetNodes() map[int]*Node {
	return h.nodes
}

func (h *HashRing) insertId(id int) {
	if len(h.ids) == 0 {
		h.ids = append(h.ids, id)
		return
	}
	newIdPos := findInsertIndex(h.ids, id) + 1
	newIds := make([]int, 0)
	newIds = append(newIds, h.ids[:newIdPos]...) // append elements BEFORE new Id
	newIds = append(newIds, id)                  // append Id
	newIds = append(newIds, h.ids[newIdPos:]...) // append elements AFTER new Id
	h.ids = newIds
}

// Example: arr = [1, 3, 8, 11, 19], target = 12, result = 3
// Example: arr = [1, 3, 8, 11, 19], target = 18, result = 3
func findInsertIndex(arr []int, target int) int {
	n := len(arr)

	// base cases
	if n == 0 {
		return 0
	}
	if target <= arr[0] {
		return 0
	}
	if target >= arr[n-1] {
		return n - 1
	}

	// well... binary search
	low := 0
	high := n - 1
	mid := 0
	for low < high {
		mid = (low + high) / 2

		if arr[mid] == target {
			return mid
		}

		if target < arr[mid] {
			if mid > 0 && target > arr[mid-1] {
				return mid - 1
			}
			high = mid - 1
		} else if target > arr[mid] {
			if mid < n-1 && target < arr[mid+1] {
				return mid
			}
			low = mid + 1
		}
	}
	return mid
}

func findClosestElement(arr []int, target int) int {
	n := len(arr)

	// base case
	if n == 0 {
		return 0
	}
	if target <= arr[0] {
		return arr[0]
	}
	if target >= arr[n-1] {
		return arr[n-1]
	}

	// well... binary search AGAIN
	low := 0
	high := n - 1
	mid := 0
	for low < high {
		mid = (low + high) / 2

		if arr[mid] == target {
			return arr[mid]
		}

		if target < arr[mid] {
			if mid > 0 && target > arr[mid-1] {
				return getClosest(target, arr[mid-1], arr[mid])
			}
			high = mid - 1
		} else if target > arr[mid] {
			if mid < n-1 && target < arr[mid+1] {
				return getClosest(target, arr[mid], arr[mid+1])
			}
			low = mid + 1
		}
	}
	return mid
}

func getClosest(target, val1, val2 int) int {
	if target-val1 >= val2-target {
		return val2
	}
	return val1
}
