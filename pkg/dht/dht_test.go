package dht

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestGetNodeForKey(t *testing.T) {
	addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:42069")
	assert.NoError(t, err, "error in resolving tcp address")
	addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.1:42070")
	assert.NoError(t, err, "error in resolving tcp address")
	addr3, err := net.ResolveTCPAddr("tcp", "127.0.0.1:42071")
	assert.NoError(t, err, "error in resolving tcp address")

	node1 := NewNode(addr1)
	node2 := NewNode(addr2)
	node3 := NewNode(addr3)

	hashRing := NewHashRing()
	hashRing.AddNode(node1, node2, node3)

	assert.NotEmptyf(t, hashRing.ids, "hash ring ids is empty")
	assert.NotEmptyf(t, hashRing.nodes, "hash ring nodes is empty")

	expectedLength := 3
	assert.Equal(t, expectedLength, len(hashRing.ids), "wrong ids length")
	assert.Equal(t, expectedLength, len(hashRing.nodes), "wrong ids length")

	testKey := "some_random_key"
	_, err = hashRing.GetNodeForKey(testKey)
	assert.NoError(t, err, "error occurred while getting node foo key", testKey)
}

func TestFindInsertIndexEmptyArr(t *testing.T) {
	arr := make([]int, 0)
	target := 12
	result := findIndex(arr, target)
	assert.Zero(t, result)
}

func TestFindInsertIndex(t *testing.T) {
	arr := []int{1, 4, 7, 12, 13, 17}
	target := 10
	expected := 2 // index
	result := findIndex(arr, target)
	assert.Equal(t, expected, result)
}

func TestFindInsertIndexDistance(t *testing.T) {
	arr1 := []int{1, 3, 8, 11, 19}
	arr2 := []int{1, 3, 8, 11, 19}
	target1 := 12
	target2 := 18

	expected := 3

	result1 := findIndex(arr1, target1)
	result2 := findIndex(arr2, target2)

	assert.Equal(t, expected, result1)
	assert.Equal(t, expected, result2)
}

func TestInsertId(t *testing.T) {
	newId := 12
	arr := []int{1, 3, 8, 11, 19, 20, 24}
	expected := []int{1, 3, 8, 11, 12, 19, 20, 24}
	hashRing := HashRing{ids: arr, nodes: nil}
	hashRing.insertId(newId)
	assert.Equal(t, expected, hashRing.ids)
}

func TestInsertIdEmptyArr(t *testing.T) {
	arr := make([]int, 0)
	newId := 69
	expected := []int{69}
	hashRing := HashRing{ids: arr, nodes: nil}
	hashRing.insertId(newId)
	assert.Equal(t, expected, hashRing.ids)
}
