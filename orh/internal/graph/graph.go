// Package graph builds and queries the execution graph of an architecture:
// which endpoint(s) each connection endpoint fans out to.
package graph

import "github.com/pertevniyalai/orh/internal/spec"

// Graph is the execution graph derived from an Architecture's connections.
type Graph struct {
	edges map[string][]string
}

// Build constructs a Graph from the architecture's connection list.
func Build(arch *spec.Architecture) *Graph {
	g := &Graph{edges: make(map[string][]string)}
	for _, c := range arch.Connections {
		g.edges[c.From] = append(g.edges[c.From], c.To)
	}
	return g
}

// Successors returns the endpoints that the given output endpoint connects
// to.
func (g *Graph) Successors(endpoint string) []string {
	return g.edges[endpoint]
}
