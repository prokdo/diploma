package graph

import (
	"context"
	"math/rand"
)

func MISGreedy[T comparable](ctx context.Context, g Graph[T]) []T {
	graph, ok := g.(*graph[T])
	if !ok {
		return nil
	}

	n := graph.size
	deleted := make([]bool, n)
	var solutionIndices []int

	initAdjMatrix(graph)
	adjMatrix := graph.cache.AdjMatrix

	for {
		if ctx.Err() != nil {
			return nil
		}

		candidates := gatherCandidates(adjMatrix, deleted)
		if len(candidates) == 0 {
			break
		}

		bestIdx := findMinDegree(adjMatrix, candidates, deleted)
		if ctx.Err() != nil {
			return nil
		}

		solutionIndices = append(solutionIndices, bestIdx)
		markDeleted(adjMatrix, bestIdx, deleted)
	}

	result := make([]T, len(solutionIndices))
	for i, idx := range solutionIndices {
		result[i] = graph.indexToVertex[idx]
	}
	return result
}

func gatherCandidates(adj *adjMatrix, deleted []bool) []int {
	var result []int
	n := adj.Size()
	for j := range n {
		if !deleted[j] {
			result = append(result, j)
		}
	}
	return result
}

func markDeleted(adj *adjMatrix, idx int, deleted []bool) {
	deleted[idx] = true
	for j := range adj.Size() {
		if adj.Get(idx, j) || adj.Get(j, idx) {
			deleted[j] = true
		}
	}
}

func findMinDegree(adj *adjMatrix, candidates []int, deleted []bool) int {
	bestDegree := adj.Size() + 1
	var minCandidates []int

	for _, idx := range candidates {
		if deleted[idx] {
			continue
		}
		degree := 0
		for j := range adj.Size() {
			if adj.Get(idx, j) && !deleted[j] {
				degree++
			}
		}
		if degree < bestDegree {
			bestDegree = degree
			minCandidates = []int{idx}
		} else if degree == bestDegree {
			minCandidates = append(minCandidates, idx)
		}
	}

	if len(minCandidates) > 0 {
		return minCandidates[rand.Intn(len(minCandidates))]
	}
	return -1
}
