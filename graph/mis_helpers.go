package graph

import "slices"

func computeCardinality(x []bool) int {
	count := 0
	for _, b := range x {
		if b {
			count++
		}
	}
	return count
}

func ComputeF1Factor[T comparable](sample, solution []T) float64 {
	if len(sample) == 0 || len(solution) == 0 {
		return 0
	}

	count := 0
	for _, v := range solution {
		if slices.Contains(sample, v) {
			count++
		}
	}

	precision := float64(count) / float64(len(solution))
	recall := float64(count) / float64(len(sample))

	denominator := precision + recall
	if denominator == 0 {
		return 0
	}

	return 2 * precision * recall / denominator
}
