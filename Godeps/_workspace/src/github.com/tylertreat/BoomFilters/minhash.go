package boom

import (
	"math"
	"math/rand"
)

// MinHash is a variation of the technique for estimating similarity between
// two sets as presented by Broder in On the resemblance and containment of
// documents:
//
// http://gatekeeper.dec.com/ftp/pub/dec/SRC/publications/broder/positano-final-wpnums.pdf
//
// This can be used to cluster or compare documents by splitting the corpus
// into a bag of words. MinHash returns the approximated similarity ratio of
// the two bags. The similarity is less accurate for very small bags of words.
func MinHash(bag1, bag2 []string) float32 {
	k := len(bag1) + len(bag2)
	hashes := make([]int, k)
	for i := 0; i < k; i++ {
		a := uint(rand.Int())
		b := uint(rand.Int())
		c := uint(rand.Int())
		x := computeHash(a*b*c, a, b, c)
		hashes[i] = int(x)
	}

	bitMap := bitMap(bag1, bag2)
	minHashValues := hashBuckets(2, k)
	minHash(bag1, 0, minHashValues, bitMap, k, hashes)
	minHash(bag2, 1, minHashValues, bitMap, k, hashes)
	return similarity(minHashValues, k)
}

func minHash(bag []string, bagIndex int, minHashValues [][]int,
	bitArray map[string][]bool, k int, hashes []int) {
	index := 0
	for element := range bitArray {
		for i := 0; i < k; i++ {
			if contains(bag, element) {
				hindex := hashes[index]
				if hindex < minHashValues[bagIndex][index] {
					minHashValues[bagIndex][index] = hindex
				}
			}
		}
		index++
	}
}

func contains(bag []string, element string) bool {
	for _, e := range bag {
		if e == element {
			return true
		}
	}
	return false
}

func bitMap(bag1, bag2 []string) map[string][]bool {
	bitArray := map[string][]bool{}
	for _, element := range bag1 {
		bitArray[element] = []bool{true, false}
	}

	for _, element := range bag2 {
		if _, ok := bitArray[element]; ok {
			bitArray[element] = []bool{true, true}
		} else if _, ok := bitArray[element]; !ok {
			bitArray[element] = []bool{false, true}
		}
	}

	return bitArray
}

func hashBuckets(numSets, k int) [][]int {
	minHashValues := make([][]int, numSets)
	for i := 0; i < numSets; i++ {
		minHashValues[i] = make([]int, k)
	}

	for i := 0; i < numSets; i++ {
		for j := 0; j < k; j++ {
			minHashValues[i][j] = math.MaxInt32
		}
	}
	return minHashValues
}

func computeHash(x, a, b, u uint) uint {
	return (a*x + b) >> (32 - u)
}

func similarity(minHashValues [][]int, k int) float32 {
	identicalMinHashes := 0
	for i := 0; i < k; i++ {
		if minHashValues[0][i] == minHashValues[1][i] {
			identicalMinHashes++
		}
	}

	return (1.0 * float32(identicalMinHashes)) / float32(k)
}
