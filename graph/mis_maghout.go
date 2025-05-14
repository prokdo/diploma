package graph

import (
	"context"
	"sync/atomic"
)

func MISMaghout[T comparable](ctx context.Context, g Graph[T]) []T {
	graph, ok := g.(*graph[T])
	if !ok {
		return nil
	}

	if graph.cache.AdjMatrix == nil {
		initAdjMatrix(graph)
	}

	n := graph.size
	bestCount := int32(-1)
	var bestSolution atomic.Value
	temp := make([]bool, n)
	bestSolution.Store(&temp)

	current := make([]bool, n)
	backtrack(ctx, graph, current, 0, &bestCount, &bestSolution)

	solution := bestSolution.Load().(*[]bool)
	var result []T
	for i, bit := range *solution {
		if bit {
			result = append(result, graph.indexToVertex[i])
		}
	}
	return result
}

func backtrack[T comparable](
	ctx context.Context,
	g *graph[T],
	current []bool,
	i int,
	bestCount *int32,
	bestSolution *atomic.Value,
) {
	if ctx.Err() != nil {
		return
	}

	if i == len(current) {
		if checkSolution(g, current) {
			count := int32(computeCardinality(current))
			currBest := atomic.LoadInt32(bestCount)

			if count > currBest {
				newCopy := make([]bool, len(current))
				copy(newCopy, current)

				bestSolution.Store(&newCopy)
				atomic.StoreInt32(bestCount, count)
			}
		}
		return
	}

	current[i] = false
	backtrack(ctx, g, current, i+1, bestCount, bestSolution)

	if ctx.Err() != nil {
		return
	}

	current[i] = true
	backtrack(ctx, g, current, i+1, bestCount, bestSolution)
}
