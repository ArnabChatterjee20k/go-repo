package main

import (
	"slices"
	"testing"
)

func TestMatch(t *testing.T) {
	trie := CreateTrie()
	trie.Add("sport/tennis/player1", 1) // exact
	trie.Add("sport/+/player1", 2)      // single-level wildcard
	trie.Add("sport/#", 3)              // multi-level wildcard
	trie.Add("+", 4)                    // matches ONE top-level segment only

	cases := []struct {
		topic string
		want  []int
	}{
		{"sport/tennis/player1", []int{1, 2, 3}}, // exact + '+' + '#'
		{"sport/golf/player1", []int{2, 3}},      // '+' + '#'
		{"sport/tennis/player1/x", []int{3}},     // only '#'; '+' must NOT span '/'
		{"sport", []int{3, 4}},                   // '#' matches the parent; top-level '+' matches one segment
		{"weather", []int{4}},                    // top-level '+' matches one segment
		{"weather/london", []int{}},              // '+' must NOT match two segments
	}

	for _, c := range cases {
		got := trie.Match(c.topic)
		slices.Sort(got)
		want := slices.Clone(c.want)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("Match(%q) = %v; want %v", c.topic, got, want)
		}
	}
}

func TestAddStoresIdAtLeafOnly(t *testing.T) {
	trie := CreateTrie()
	trie.Add("sport/tennis/player1", 7)

	leaf := trie.root.get("sport").get("tennis").get("player1")
	if leaf == nil || !leaf.hasId(7) {
		t.Fatalf("id 7 should live at the leaf sport/tennis/player1")
	}
	// the id must NOT leak onto intermediate nodes
	if trie.root.get("sport").hasId(7) {
		t.Errorf("id 7 leaked onto the intermediate 'sport' node")
	}
}

func TestRemove(t *testing.T) {
	trie := CreateTrie()
	trie.Add("a/b", 1)

	ok, err := trie.Remove("a/b", 1)
	if !ok || err != nil {
		t.Fatalf("Remove(a/b, 1) = %v, %v; want true, nil", ok, err)
	}
	if trie.root.get("a").get("b").hasId(1) {
		t.Errorf("id 1 still present after Remove")
	}

	// removing an unknown topic reports an error
	if _, err := trie.Remove("x/y", 1); err == nil {
		t.Errorf("expected an error removing an unknown topic")
	}
}

func TestRemoveTopic(t *testing.T) {
	trie := CreateTrie()
	trie.Add("sport/football/p3", 3)
	trie.Add("sport/tennis/p1", 1)

	trie.RemoveTopic("sport/football")

	if trie.root.get("sport").get("football") != nil {
		t.Errorf("sport/football should be gone")
	}
	if trie.root.get("sport").get("tennis") == nil {
		t.Errorf("sport/tennis should be untouched")
	}
}
