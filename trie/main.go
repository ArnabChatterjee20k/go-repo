package main

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
)

const ROOT string = "ROOT"

type Node struct {
	nodes map[string]*Node
	ids   map[int]int
}

func newNode() *Node {
	return &Node{
		nodes: map[string]*Node{},
		ids:   map[int]int{},
	}
}

func (root *Node) add(topic string, node *Node) {
	root.nodes[topic] = node
}

func (root *Node) addId(id int) {
	root.ids[id] = 1
}

func (root *Node) get(topic string) *Node {
	if node, ok := root.nodes[topic]; ok {
		return node
	}
	return nil
}

func (root *Node) hasId(id int) bool {
	_, ok := root.ids[id]
	return ok
}

func (root *Node) remove(topic string) {
	delete(root.nodes, topic)
}

func (root *Node) removeId(id int) {
	delete(root.ids, id)
}

type Trie struct {
	root Node
}

func (trie *Trie) Add(topic string, id int) *Node {
	split := strings.Split(topic, "/")
	node := &trie.root
	for _, level := range split {
		child := node.get(level)
		if child == nil {
			child = newNode()
			node.add(level, child)
		}
		node = child
	}
	node.addId(id)
	return node
}

func (trie *Trie) Remove(topic string, id int) (bool, error) {
	split := strings.Split(topic, "/")
	node := &trie.root
	for _, level := range split {
		child := node.get(level)
		if child == nil {
			return false, errors.New("invalid topic")
		}
		node = child
	}
	node.removeId(id) // remove only at the leaf, where the id was added
	return true, nil
}

func (trie *Trie) RemoveTopic(topic string) {
	split := strings.Split(topic, "/")
	node := &trie.root
	for _, level := range split[:len(split)-1] {
		child := node.get(level)
		if child == nil {
			return
		}
		node = child
	}
	node.remove(split[len(split)-1])
}

func collect(ids map[int]int, newIds map[int]int) {
	for id := range newIds {
		ids[id] = 1
	}
}

func (trie *Trie) Match(topic string) []int {
	ids := map[int]int{}
	split := strings.Split(topic, "/")
	current := []*Node{&trie.root}
	for _, topic := range split {
		next := []*Node{}
		for _, root := range current {
			if node := root.get(topic); node != nil {
				next = append(next, node)
			}

			if node := root.get("#"); node != nil {
				collect(ids, node.ids)
			}

			if node := root.get("+"); node != nil {
				next = append(next, node)
			}
		}
		current = next
	}
	for _, node := range current {
		collect(ids, node.ids)
		if node := node.get("#"); node != nil {
			collect(ids, node.ids)
		}
	}
	return slices.Collect(maps.Keys(ids))
}

func CreateTrie() Trie {
	return Trie{root: *newNode()}
}

func main() {
	trie := CreateTrie()
	trie.Add("sport/tennis/player1", 1)
	trie.Add("sport/tennis/player2", 2)
	trie.Add("sport/football/player3", 3)

	leaf := trie.root.get("sport").get("tennis").get("player1")
	fmt.Println("player1 has id 1:", leaf.hasId(1))

	trie.Remove("sport/tennis/player1", 1)
	fmt.Println("after remove, has id 1:", leaf.hasId(1))

	trie.RemoveTopic("sport/football")
	fmt.Println("football subtree gone:", trie.root.get("sport").get("football") == nil)
}
