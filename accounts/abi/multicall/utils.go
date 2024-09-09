package multicall

func mapArray[A any, B any](coll []A, mapper func(i A, index uint64) B) []B {
	out := make([]B, len(coll))
	for i, item := range coll {
		out[i] = mapper(item, uint64(i))
	}
	return out
}
