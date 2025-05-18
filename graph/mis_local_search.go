package graph

import (
	"context"
	"slices"
)

func MISLocalSearch[T comparable](ctx context.Context, g Graph[T], genome []bool, localIters int) []bool {
	graph, ok := g.(*graph[T])
	if !ok {
		return nil
	}

	initAdjMatrix(graph)
	adj := graph.cache.AdjMatrix
	best := slices.Clone(genome)
	bestFitness := computeCardinality(best)
	n := len(genome)

	for range localIters {
		if ctx.Err() != nil {
			return nil
		}

		improved := false
		for j := range n {
			temp := slices.Clone(best)
			temp[j] = !temp[j]

			if checkSolutionFast(adj, temp, j) {
				currentFitness := computeCardinality(temp)
				if currentFitness > bestFitness {
					best = temp
					bestFitness = currentFitness
					improved = true
				}
			}
		}

		if !improved {
			break
		}
	}
	return best
}

func checkSolutionFast(adj *adjMatrix, genome []bool, flippedIdx int) bool {
	for j := range adj.Size() {
		if genome[j] && (adj.Get(flippedIdx, j) || adj.Get(j, flippedIdx)) {
			return false
		}
	}
	return true
}
