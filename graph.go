package main

import (
	"container/heap"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
)

const GRAPH_N = 16
const GRAPH_K = 5

// Gid identify nodes or edges
type Gid uint32

// Node represents one node of the graph (1 airport)
type Node struct {
	id   Gid
	loc  string
	from []Gid
	to   []Gid
}

// Edge represents a direct route between two airports
type Edge struct {
	id     Gid
	weight float32
	from   Gid
	to     Gid
}

// Path contains the nodes and edges of a complete route
type Path struct {
	nodes  []Gid
	edges  []Gid
	weight float32
}

// NewPath allocates a Path object
func NewPath() *Path {
	return &Path{
		nodes: make([]Gid, 0, GRAPH_N),
		edges: make([]Gid, 0, GRAPH_N),
	}
}

// reset initializes a new path from a location
func (p *Path) reset(src Gid) {
	p.nodes = p.nodes[:1]
	p.nodes[0] = src
	p.edges = p.edges[:0]
}

// add a new leg to the path
func (p *Path) add(n, e Gid) {
	p.nodes = append(p.nodes, n)
	p.edges = append(p.edges, e)
}

// pop removes the last leg of the path
func (p *Path) pop() (Gid, Gid) {
	n, e := len(p.nodes)-1, len(p.edges)-1
	nid, eid := p.nodes[n], p.edges[e]
	p.nodes, p.edges = p.nodes[:n], p.edges[:e]
	return nid, eid
}

// contains checks a given node is already in the path or not
func (p *Path) contains(n Gid) bool {
	for _, x := range p.nodes {
		if n == x {
			return true
		}
	}
	return false
}

// set assigns data from an other path
func (p *Path) set(other *Path) {
	p.weight = 0.0
	p.nodes = append(p.nodes[:0], other.nodes...)
	p.edges = append(p.edges[:0], other.edges...)
}

// pathPool is used to amortize the allocation cost of the path objects
var pathPool = sync.Pool{
	New: func() interface{} {
		return NewPath()
	},
}

// PathHeap is a heap of Path object
type PathHeap []*Path

// NewPathHeap allocates a new PathHeap object
func NewPathHeap(k int) *PathHeap {
	ph := PathHeap(make([]*Path, 0, k+1))
	return &ph
}

// Len returns the size of the heap
func (ph PathHeap) Len() int {
	return len(ph)
}

// Less compares two elements in the heap
func (ph PathHeap) Less(i, j int) bool {
	return ph[i].weight > ph[j].weight
}

// Swap exchanges two elements in the heap
func (ph PathHeap) Swap(i, j int) {
	ph[i], ph[j] = ph[j], ph[i]
}

// Push adds an element in the heap
func (ph *PathHeap) Push(x interface{}) {
	*ph = append(*ph, x.(*Path))
}

// Pop removes the path with the highest weight from the heap
func (ph *PathHeap) Pop() interface{} {
	n := len(*ph) - 1
	res := (*ph)[n]
	*ph = (*ph)[0:n]
	return res
}

// Graph stores the graph data and the contextual information for a given route exploration
type Graph struct {
	nodes  []Node
	edges  []Edge
	locs   map[string]Gid
	path   *Path
	stack  []Gid
	heap   *PathHeap
	output strings.Builder
}

// NewGraph allocates a new graph object
func NewGraph() *Graph {
	return &Graph{
		nodes: make([]Node, 0, GRAPH_N),
		edges: make([]Edge, 0, GRAPH_N),
		locs:  make(map[string]Gid, GRAPH_N),
		path:  NewPath(),
		stack: make([]Gid, 0, GRAPH_N),
		heap:  NewPathHeap(GRAPH_K),
	}
}

// addNode adds a single node (airport) to the graph
func (g *Graph) addNode(loc string) {
	node := Node{
		id:  Gid(len(g.nodes)),
		loc: loc,
	}
	g.nodes = append(g.nodes, node)
	g.locs[loc] = node.id
}

// addEdge adds a direct route between two nodes
func (g *Graph) addEdge(from, to Gid, weight float32) {
	edge := Edge{
		id:     Gid(len(g.edges)),
		weight: weight,
		from:   from,
		to:     to,
	}
	g.edges = append(g.edges, edge)
	f := g.node(from)
	f.to = append(f.to, edge.id)
	t := g.node(to)
	t.from = append(t.from, edge.id)
}

// node returns a node for a given Gid
func (g *Graph) node(n Gid) *Node {
	return &g.nodes[int(n)]
}

// edge returns an edge for a given Gid
func (g *Graph) edge(e Gid) *Edge {
	return &g.edges[int(e)]
}

// search is the high-level entry point for route exploring between two locations
func (g *Graph) search(loc1, loc2 string) error {
	src, ok := g.locs[loc1]
	if !ok {
		return errors.New("not found")
	}
	dst, ok := g.locs[loc2]
	if !ok {
		return errors.New("not found")
	}
	g.allPaths(src, dst)
	g.display()
	return nil
}

// allPaths implements a non recursive DFS algorithm to generate all the routes
func (g *Graph) allPaths(src, dst Gid) {

	// Reset the path and heap
	g.path.reset(src)
	*g.heap = (*g.heap)[:0]

	// Reset the stack
	// The stack contains all the next edges to explore
	g.stack = g.stack[:0]
	g.stack = append(g.stack, g.node(src).to...)

	// We walk the graph depth-first
	var eid, nid Gid
	for len(g.stack) > 0 {

		// Pop the edge to explore
		idx := len(g.stack) - 1
		eid, g.stack = g.stack[idx], g.stack[:idx]

		// Stop when the end of edge marker is detected, and backtrack
		if eid == math.MaxUint32 {
			g.path.pop()
			continue
		}

		// Skip paths with loops
		nid = g.edge(eid).to
		if g.path.contains(nid) {
			continue
		}

		// Add the next node to the path, and push the corresponding edges to the stack
		// Do not forget to signal the end of edge using a MaxUint32 marker
		g.path.add(nid, eid)
		g.stack = append(g.stack, math.MaxUint32)
		if nid == dst {
			// We have found a relevant path: collect and process it
			g.collect()
		} else {
			// Consider all the edges of the node to be analyzed next
			g.stack = append(g.stack, g.node(nid).to...)
		}
	}
}

// collect processes a relevant path by calculating its weight and top-k filtering
func (g *Graph) collect() {

	// Copy the path, and calculate its weight
	p := pathPool.Get().(*Path)
	p.set(g.path)
	p.weight = g.calculateWeight()

	// Push it to the heap as a top-k filtering mechanism
	heap.Push(g.heap, p)
	if g.heap.Len() > GRAPH_K {
		pathPool.Put(heap.Pop(g.heap))
	}
}

// calculateWeight adds the weights of all the edges of the selected path
func (g *Graph) calculateWeight() float32 {

	var w float32
	for _, x := range g.path.edges {
		w += g.edge(x).weight
	}
	return w
}

// display builds the final display of the filtered paths
func (g *Graph) display() {

	g.output.Reset()
	g.output.WriteString("Results\n")
	for range *g.heap {
		p := heap.Pop(g.heap).(*Path)
		g.output.WriteString(g.node(p.nodes[0]).loc)
		for _, x := range p.nodes[1:] {
			g.output.WriteByte('-')
			g.output.WriteString(g.node(x).loc)
		}
		fmt.Fprintf(&g.output, " %f\n", p.weight)
		pathPool.Put(p)
	}

	// Keep it commented except if you want to debug
	//fmt.Println(g.output.String())
}

// BenchGraph is a route exploring benchmark based on a simple graph implementation.
type BenchGraph struct {
	g *Graph
}

// NewBenchGraph allocates a new benchmark object
func NewBenchGraph() *BenchGraph {
	bg := &BenchGraph{
		g: NewGraph(),
	}
	// Totally random locations
	for _, x := range []string{"AMS", "DUB", "CDG", "FRA", "NCE", "MRS", "LON", "MUC", "BOS", "JFK", "LAX", "SFO"} {
		bg.g.addNode(x)
	}
	// Totally random direct routes
	for _, x := range []struct {
		from, to int
		dist     float32
	}{
		{0, 1, 10},
		{1, 2, 20},
		{2, 0, 5},
		{7, 6, 12},
		{2, 3, 8},
		{3, 4, 13},
		{4, 5, 2},
		{5, 7, 3},
		{7, 0, 20},
		{7, 2, 30},
		{1, 4, 5},
		{3, 6, 40},
		{2, 8, 200},
		{6, 8, 220},
		{2, 9, 210},
		{6, 9, 215},
		{0, 9, 212},
		{8, 9, 10},
		{8, 10, 90},
		{9, 10, 95},
		{7, 8, 230},
		{0, 6, 25},
		{10, 11, 10},
		{10, 9, 98},
	} {
		bg.g.addEdge(Gid(x.from), Gid(x.to), x.dist)
		bg.g.addEdge(Gid(x.to), Gid(x.from), x.dist)
	}
	return bg
}

// We will explore routes for the following pairs of locations
var GraphLocations = []string{
	"AMS", "LON",
	"LAX", "NCE",
	"MRS", "DUB",
	"SFO", "FRA",
}

// Run the graph benchmark
func (bg *BenchGraph) Run() {

	// Explore routes for the pairs of locations
	for i := 0; i < len(GraphLocations); i += 2 {
		if err := bg.g.search(GraphLocations[i], GraphLocations[i+1]); err != nil {
			log.Fatal(err)
		}
	}
}
