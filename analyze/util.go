package analyze

import "math"

func incrementUint64(n *uint64) {
	if *n != math.MaxUint64 {
		*n++
	}
}

func safeAddUint64(a, b uint64) uint64 {
	sum := a + b
	if sum >= a {
		return sum
	}
	return math.MaxUint64
}
