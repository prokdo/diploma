package graph

import (
	"context"
	"sync"
	"sync/atomic"
)

func MISMaghout[T comparable](ctx context.Context, g Graph[T], parallelDepth int) []T {
	graph, ok := g.(*graph[T])
	if !ok || graph == nil {
		return nil
	}

	if graph.cache.AdjMatrix == nil {
		initAdjMatrix(graph)
	}

	n := graph.size
	if n == 0 {
		return nil
	}

	if parallelDepth <= 0 || parallelDepth > n {
		parallelDepth = n
	}
	if parallelDepth > 16 {
		parallelDepth = 16
	}

	total := 1 << parallelDepth

	var wg sync.WaitGroup
	var bestCount int32 = -1
	var bestSolution atomic.Value
	bestSolution.Store(make([]bool, n))

	for mask := range total {
		mask := mask
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			current := make([]bool, n)
			for i := range parallelDepth {
				current[i] = (mask>>i)&1 == 1
			}

			backtrack(ctx, graph, current, parallelDepth, &bestCount, &bestSolution)
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil
	}

	solution := bestSolution.Load().([]bool)
	result := make([]T, 0, n)
	for i, include := range solution {
		if include {
			result = append(result, graph.indexToVertex[i])
		}
	}
	return result
}

func backtrack[T comparable](
	ctx context.Context,
	g *graph[T],
	current []bool,
	idx int,
	bestCount *int32,
	bestSolution *atomic.Value,
) {
	if ctx.Err() != nil {
		return
	}

	if idx == len(current) {
		if checkSolution(g, current) {
			count := int32(computeCardinality(current))
			if count > atomic.LoadInt32(bestCount) {
				newSol := make([]bool, len(current))
				copy(newSol, current)
				atomic.StoreInt32(bestCount, count)
				bestSolution.Store(newSol)
			}
		}
		return
	}

	current[idx] = false
	backtrack(ctx, g, current, idx+1, bestCount, bestSolution)

	current[idx] = true
	backtrack(ctx, g, current, idx+1, bestCount, bestSolution)

}

func checkSolution[T comparable](g *graph[T], x []bool) bool {
	for i := range x {
		if x[i] {
			for j := i + 1; j < len(x); j++ {
				if x[j] && (g.cache.AdjMatrix.Get(i, j) || g.cache.AdjMatrix.Get(j, i)) {
					return false
				}
			}
		}
	}
	return true
}
