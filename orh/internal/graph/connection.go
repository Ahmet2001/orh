package graph

import "strings"

// SplitEndpoint splits a connection endpoint like "agent.input" into its
// component and port. A bare endpoint (a graph boundary node such as
// "input"/"output", or a component name with no explicit port) returns an
// empty port. The split happens on the *last* dot, not the first, because a
// component name produced by composition (see internal/composition) is
// itself dotted, e.g. "search.crawler.input" is component "search.crawler"
// port "input".
func SplitEndpoint(endpoint string) (component, port string) {
	if idx := strings.LastIndex(endpoint, "."); idx != -1 {
		return endpoint[:idx], endpoint[idx+1:]
	}
	return endpoint, ""
}
