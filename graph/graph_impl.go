package graph

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"sync"
)

type graph[T comparable] struct {
	Graph[T]

	gtype GraphType
	size  int

	adjList map[T][]T

	cache graphCache

	hasLoop   *bool
	isAcyclic *bool

	indexToVertex []T
	vertexToIndex map[T]int
}

func (g *graph[T]) Type() GraphType {
	return g.gtype
}

func (g *graph[T]) Size() int {
	return g.size
}

func (g *graph[T]) HasLoop() bool {
	if g.size == 0 {
		return false
	}

	if g.hasLoop != nil {
		return *g.hasLoop
	}

	g.hasLoop = new(bool)
	if g.cache.AdjMatrix != nil {
		*g.hasLoop = checkForLoopsByAdjMatrix(*g.cache.AdjMatrix)
	} else {
		*g.hasLoop = checkForLoopsByAdjList(g.adjList)
	}

	return *g.hasLoop
}

func (g *graph[T]) HasWeights() bool {
	return g.cache.WeightMatrix != nil
}

func (g *graph[T]) IsAcyclic() bool {
	if g.size == 0 {
		return true
	}

	if g.isAcyclic != nil {
		return *g.isAcyclic
	}

	g.isAcyclic = new(bool)
	*g.isAcyclic = !checkForCycles(g)
	return *g.isAcyclic
}

func (g *graph[T]) AddEdge(from, to *T, weight ...float64) bool {
	if from == nil || to == nil {
		return false
	}

	if g.ContainsEdge(from, to) {
		return false
	}

	if !g.ContainsVertex(from) {
		g.AddVertex(from)
	}

	if !g.ContainsVertex(to) {
		g.AddVertex(to)
	}

	g.adjList[*from] = append(g.adjList[*from], *to)
	if g.gtype == Undirected {
		g.adjList[*to] = append(g.adjList[*to], *from)
	}

	if g.cache.AdjMatrix != nil {
		fromIdx := g.vertexToIndex[*from]
		toIdx := g.vertexToIndex[*to]
		g.cache.AdjMatrix.Set(fromIdx, toIdx, true)
		if g.gtype == Undirected {
			g.cache.AdjMatrix.Set(toIdx, fromIdx, true)
		}
	}

	if len := len(weight); len != 0 {
		if g.cache.WeightMatrix == nil {
			initWeightMatrix(g)
		}
		fromIdx := g.vertexToIndex[*from]
		toIdx := g.vertexToIndex[*to]
		g.cache.WeightMatrix.Set(fromIdx, toIdx, weight[len-1])
		if g.gtype == Undirected {
			g.cache.WeightMatrix.Set(toIdx, fromIdx, weight[len-1])
		}
	}

	if *from == *to {
		g.hasLoop = new(bool)
		*g.hasLoop = true
	}

	g.isAcyclic = nil

	return true

}

func (g *graph[T]) AddVertex(v *T) bool {
	if v == nil {
		return false
	}

	if g.ContainsVertex(v) {
		return false
	}

	g.vertexToIndex[*v] = len(g.indexToVertex)
	g.indexToVertex = append(g.indexToVertex, *v)

	g.adjList[*v] = []T{}

	g.size++

	return true
}

func (g *graph[T]) ClearCache() {
	g.cache = graphCache{}
}

func (g *graph[T]) ClearWeights() {
	g.cache.WeightMatrix = nil
}

func (g *graph[T]) ContainsEdge(from, to *T) bool {
	if from == nil || to == nil {
		return false
	}

	if _, ok := g.vertexToIndex[*from]; !ok {
		return false
	}

	if _, ok := g.vertexToIndex[*to]; !ok {
		return false
	}

	if g.cache.AdjMatrix == nil {
		if slices.Contains(g.adjList[*from], *to) {
			return true
		}
	} else {
		fromIdx := g.vertexToIndex[*from]
		toIdx := g.vertexToIndex[*to]

		return g.cache.AdjMatrix.Get(fromIdx, toIdx)
	}

	return false
}

func (g *graph[T]) ContainsVertex(v *T) bool {
	if v == nil {
		return false
	}

	_, ok := g.vertexToIndex[*v]
	return ok
}

func (g *graph[T]) ExportToFile(path string, overwrite bool, verticesToColor ...[]T) error {
	_, err := os.Stat(path)
	if err == nil {
		if !overwrite {
			return os.ErrExist
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	dotStr := g.Dot(verticesToColor...)

	err = os.WriteFile(path, []byte(dotStr), 0644)
	if err != nil {
		return err
	}
	return nil
}

func (g *graph[T]) GetAllEdges() []*EdgeOptions[T] {
	resultChan := make(chan *EdgeOptions[T], 100)
	var wg sync.WaitGroup

	for from, neighbors := range g.adjList {
		from := from
		neighbors := neighbors

		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, to := range neighbors {
				to := to
				var weight float64
				if g.cache.WeightMatrix != nil {
					weight = g.cache.WeightMatrix.Get(g.vertexToIndex[from], g.vertexToIndex[to])
				} else {
					weight = NO_WEIGHT
				}
				resultChan <- &EdgeOptions[T]{
					From:   &from,
					To:     &to,
					Weight: weight,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	edges := make([]*EdgeOptions[T], 0)
	for edge := range resultChan {
		edges = append(edges, edge)
	}

	return edges
}

func (g *graph[T]) GetAllVertices() []T {
	result := make([]T, len(g.indexToVertex))
	copy(result, g.indexToVertex)
	return result
}

func (g *graph[T]) GetEdge(from, to *T) *EdgeOptions[T] {
	if from == nil || to == nil {
		return nil
	}

	if g.cache.AdjMatrix == nil {
		initAdjMatrix(g)
	}

	if !g.ContainsEdge(from, to) {
		return nil
	}

	fromIdx := g.vertexToIndex[*from]
	toIdx := g.vertexToIndex[*to]

	if !g.cache.AdjMatrix.Get(fromIdx, toIdx) {
		return nil
	}
	if g.cache.WeightMatrix != nil {
		return &EdgeOptions[T]{
			From:   from,
			To:     to,
			Weight: g.cache.WeightMatrix.Get(fromIdx, toIdx),
		}
	}
	return &EdgeOptions[T]{
		From:   from,
		To:     to,
		Weight: NO_WEIGHT,
	}
}

func (g *graph[T]) GetEdgesOf(v *T) []*EdgeOptions[T] {
	if v == nil || !g.ContainsVertex(v) {
		return nil
	}

	neighbors := g.adjList[*v]
	resultChan := make(chan *EdgeOptions[T], len(neighbors)*2)
	var wg sync.WaitGroup

	for _, to := range neighbors {
		to := to
		wg.Add(1)
		go func() {
			defer wg.Done()

			var weight float64
			if g.cache.WeightMatrix != nil {
				weight = g.cache.WeightMatrix.Get(g.vertexToIndex[*v], g.vertexToIndex[to])
			} else {
				weight = NO_WEIGHT
			}

			resultChan <- &EdgeOptions[T]{
				From:   v,
				To:     &to,
				Weight: weight,
			}

			if g.gtype == Undirected {
				resultChan <- &EdgeOptions[T]{
					From:   &to,
					To:     v,
					Weight: weight,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	edges := make([]*EdgeOptions[T], 0, len(neighbors)*2)
	for edge := range resultChan {
		edges = append(edges, edge)
	}

	return edges
}

func (g *graph[T]) GetNeighbors(v *T) []T {
	if v == nil || !g.ContainsVertex(v) {
		return nil
	}

	if g.cache.AdjMatrix == nil {
		initAdjMatrix(g)
	}

	vIdx := g.vertexToIndex[*v]
	resultChan := make(chan T, len(g.indexToVertex)*2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := range g.indexToVertex {
			if g.cache.AdjMatrix.Get(i, vIdx) {
				vertex := g.indexToVertex[i]
				resultChan <- vertex
			}
		}
	}()

	if g.gtype == Directed {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range g.indexToVertex {
				if g.cache.AdjMatrix.Get(vIdx, i) {
					vertex := g.indexToVertex[i]
					resultChan <- vertex
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	neighbors := make([]T, 0)
	for v := range resultChan {
		neighbors = append(neighbors, v)
	}

	return neighbors
}

func (g *graph[T]) GetWeight(from, to *T) (float64, bool) {
	if from == nil || to == nil {
		return NO_WEIGHT, false
	}

	if !g.ContainsEdge(from, to) {
		return NO_WEIGHT, false
	}

	if g.cache.WeightMatrix == nil {
		return NO_WEIGHT, true
	}

	fromIdx := g.vertexToIndex[*from]
	toIdx := g.vertexToIndex[*to]

	return g.cache.WeightMatrix.Get(fromIdx, toIdx), true
}

func (g *graph[T]) RemoveEdge(from, to *T) bool {
	if from == nil || to == nil {
		return false
	}

	var removed bool
	var wg sync.WaitGroup

	if neighbors, exists := g.adjList[*from]; exists {
		for i, neighbor := range neighbors {
			if neighbor == *to {
				g.adjList[*from] = slices.Delete(neighbors, i, i+1)
				removed = true
				break
			}
		}
	}

	if g.gtype == Undirected {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if neighbors, exists := g.adjList[*to]; exists {
				for i, neighbor := range neighbors {
					if neighbor == *from {
						g.adjList[*to] = slices.Delete(neighbors, i, i+1)
						break
					}
				}
			}
		}()
	}

	if g.cache.AdjMatrix != nil {
		fromIdx := g.vertexToIndex[*from]
		toIdx := g.vertexToIndex[*to]

		wg.Add(1)
		go func() {
			defer wg.Done()
			g.cache.AdjMatrix.Set(fromIdx, toIdx, false)
		}()

		if g.gtype == Undirected {
			wg.Add(1)
			go func() {
				defer wg.Done()
				g.cache.AdjMatrix.Set(toIdx, fromIdx, false)
			}()
		}
	}

	if g.cache.WeightMatrix != nil {
		fromIdx := g.vertexToIndex[*from]
		toIdx := g.vertexToIndex[*to]

		wg.Add(1)
		go func() {
			defer wg.Done()
			g.cache.WeightMatrix.Set(fromIdx, toIdx, NO_WEIGHT)
		}()

		if g.gtype == Undirected {
			wg.Add(1)
			go func() {
				defer wg.Done()
				g.cache.WeightMatrix.Set(toIdx, fromIdx, NO_WEIGHT)
			}()
		}
	}

	wg.Wait()
	return removed
}

func (g *graph[T]) RemoveVertex(v *T) bool {
	if v == nil || !g.ContainsVertex(v) {
		return false
	}

	index := g.vertexToIndex[*v]

	delete(g.adjList, *v)

	delete(g.vertexToIndex, *v)
	g.indexToVertex = slices.Delete(g.indexToVertex, index, index+1)
	g.size--

	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, vertex := range g.indexToVertex {
		wg.Add(1)
		go func(i int, vertex T) {
			defer wg.Done()
			mu.Lock()
			g.vertexToIndex[vertex] = i
			mu.Unlock()
		}(i, vertex)
	}
	wg.Wait()

	for from := range g.adjList {
		newNeighbors := make([]T, 0, len(g.adjList[from]))
		for _, neighbor := range g.adjList[from] {
			if neighbor != *v {
				newNeighbors = append(newNeighbors, neighbor)
			}
		}
		g.adjList[from] = newNeighbors
	}

	if g.cache.AdjMatrix != nil {
		newAdjMatrix := newAdjMatrix(g.size)
		reduceMatrix(g.cache.AdjMatrix, newAdjMatrix, index)
		g.cache.AdjMatrix = newAdjMatrix
	}

	if g.cache.WeightMatrix != nil {
		newWeightMatrix := newWeightMatrix(g.size)
		reduceMatrix(g.cache.WeightMatrix, newWeightMatrix, index)
		g.cache.WeightMatrix = newWeightMatrix
	}

	return true
}

func (g *graph[T]) Reset() {
	g.size = 0
	g.adjList = make(map[T][]T)
	g.cache = graphCache{}
	g.hasLoop = nil
	g.isAcyclic = nil
	g.indexToVertex = make([]T, 0)
	g.vertexToIndex = make(map[T]int)
}

func (g *graph[T]) SetWeight(from, to *T, weight float64) bool {
	if from == nil || to == nil {
		return false
	}

	if !g.ContainsEdge(from, to) {
		return false
	}

	if g.cache.WeightMatrix == nil {
		initWeightMatrix(g)
	}

	fromIdx := g.vertexToIndex[*from]
	toIdx := g.vertexToIndex[*to]

	g.cache.WeightMatrix.Set(fromIdx, toIdx, weight)

	if g.gtype == Undirected {
		g.cache.WeightMatrix.Set(toIdx, fromIdx, weight)
	}

	return true
}

func (g *graph[T]) Dot(verticesToColor ...[]T) string {
	var solution []T
	if len(verticesToColor) > 0 {
		solution = verticesToColor[0]
	}

	var buf bytes.Buffer

	if g.gtype == Directed {
		buf.WriteString("digraph {\n")
	} else {
		buf.WriteString("graph {\n")
	}

	buf.WriteString("layout=circo;\n")

	for _, v := range g.GetAllVertices() {
		if slices.Contains(solution, v) {
			buf.WriteString(fmt.Sprintf("  %v [color=red, style=filled, shape=circle];\n", v))
		} else {
			buf.WriteString(fmt.Sprintf("  %v [shape=circle];\n", v))
		}
	}

	var connector string
	if g.gtype == Directed {
		connector = "->"
	} else {
		connector = "--"
	}

	seenEdges := make(map[[2]string]bool)

	for from, neighbors := range g.adjList {
		fromStr := fmt.Sprintf("%v", from)

		for _, to := range neighbors {
			toStr := fmt.Sprintf("%v", to)

			if g.gtype == Undirected {
				key := [2]string{fromStr, toStr}
				if fromStr > toStr {
					key = [2]string{toStr, fromStr}
				}
				if seenEdges[key] {
					continue
				}
				seenEdges[key] = true
			}

			edgeStr := fmt.Sprintf("  %v %s %v", from, connector, to)

			if g.cache.WeightMatrix != nil {
				fromIdx, _ := g.vertexToIndex[from]
				toIdx, _ := g.vertexToIndex[to]
				weight := g.cache.WeightMatrix.Get(fromIdx, toIdx)
				if weight != NO_WEIGHT {
					edgeStr += fmt.Sprintf(" [weight=%v]", weight)
				}
			}
			edgeStr += ";\n"

			buf.WriteString(edgeStr)
		}
	}

	buf.WriteString("}\n")
	return buf.String()
}
