package trace

import "sort"

// Node is a span in its parent/child tree position.
type Node struct {
	Span     Span
	Children []*Node
	Orphan   bool
}

// Tree contains roots and the spans whose stated parent was absent.
type Tree struct {
	Roots   []*Node
	Orphans []*Node
}

// BuildTree nests spans under known parents and promotes missing-parent spans
// to marked roots. Roots and children are ordered by start time.
func BuildTree(t Trace) Tree {
	nodes := make(map[string]*Node, len(t.Spans))
	for _, span := range t.Spans {
		nodes[span.SpanID] = &Node{Span: span}
	}
	tree := Tree{}
	for _, span := range t.Spans {
		node := nodes[span.SpanID]
		if span.ParentSpanID == "" {
			tree.Roots = append(tree.Roots, node)
		} else if parent, ok := nodes[span.ParentSpanID]; ok && parent != node {
			parent.Children = append(parent.Children, node)
		} else {
			node.Orphan = true
			tree.Orphans = append(tree.Orphans, node)
			tree.Roots = append(tree.Roots, node)
		}
	}
	var order func([]*Node)
	order = func(nodes []*Node) {
		sort.SliceStable(nodes, func(i, j int) bool {
			return nodes[i].Span.StartedAt.Before(nodes[j].Span.StartedAt)
		})
		for _, node := range nodes {
			order(node.Children)
		}
	}
	order(tree.Roots)
	return tree
}
