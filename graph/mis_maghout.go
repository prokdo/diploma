package graph

import (
	"context"
	"sync"
	"sync/atomic"
)

func MISMaghout[T comparable](ctx context.Context, g Graph[T], parallelDepth int) []T {
	if parallelDepth <= 0 {
		return nil
	}

	graph, ok := g.(*graph[T])
	if !ok {
		return nil
	}

	if graph.cache.AdjMatrix == nil {
		initAdjMatrix(graph)
	}

	n := graph.size

	if parallelDepth > 20 {
		parallelDepth = 20
	}

	bestCount := int32(-1)
	var bestSolution atomic.Value
	temp := make([]bool, n)
	bestSolution.Store(&temp)

	var wg sync.WaitGroup
	total := 1 << parallelDepth

	for mask := range total {
		if ctx.Err() != nil {
			break
		}

		wg.Add(1)
		go func(mask int) {
			defer wg.Done()

			current := make([]bool, n)
			valid := true

			for i := range parallelDepth {
				if ctx.Err() != nil {
					return
				}
				current[i] = (mask>>i)&1 == 1
				if current[i] && !checkSolutionPartial(graph, current, i) {
					valid = false
					break
				}
			}

			if valid {
				backtrack(ctx, graph, current, parallelDepth, &bestCount, &bestSolution)
			}
		}(mask)
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil
	}

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
	if checkSolutionPartial(g, current, i) {
		backtrack(ctx, g, current, i+1, bestCount, bestSolution)
	}
}

func checkSolutionPartial[T comparable](g *graph[T], x []bool, i int) bool {
	if !x[i] {
		return true
	}

	for j := range i {
		if x[j] && g.cache.AdjMatrix.Get(i, j) {
			return false
		}
	}
	return true
}
