package parlia

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func TestImpactOfValidatorOutOfService(t *testing.T) {
	testCases := []struct {
		totalValidators int
		downValidators  int
	}{
		{3, 1},
		{5, 2},
		{10, 1},
		{10, 4},
		{21, 1},
		{21, 3},
		{21, 5},
		{21, 10},
	}
	for _, tc := range testCases {
		simulateValidatorOutOfService(tc.totalValidators, tc.downValidators)
	}
}

func simulateValidatorOutOfService(totalValidators int, downValidators int) {
	downBlocks := 10000
	recoverBlocks := 10000
	recents := make(map[uint64]int)

	validators := make(map[int]bool, totalValidators)
	down := make([]int, totalValidators)
	for i := 0; i < totalValidators; i++ {
		validators[i] = true
		down[i] = i
	}
	rand.Shuffle(totalValidators, func(i, j int) {
		down[i], down[j] = down[j], down[i]
	})
	for i := 0; i < downValidators; i++ {
		delete(validators, down[i])
	}
	isRecentSign := func(idx int) bool {
		for _, signIdx := range recents {
			if signIdx == idx {
				return true
			}
		}
		return false
	}
	isInService := func(idx int) bool {
		return validators[idx]
	}

	downDelay := time.Duration(0)
	for h := 1; h <= downBlocks; h++ {
		if limit := uint64(totalValidators/2 + 1); uint64(h) >= limit {
			delete(recents, uint64(h)-limit)
		}
		proposer := h % totalValidators
		if !isInService(proposer) || isRecentSign(proposer) {
			candidates := make(map[int]bool, totalValidators/2)
			for v := range validators {
				if !isRecentSign(v) {
					candidates[v] = true
				}
			}
			if len(candidates) == 0 {
				panic("can not test such case")
			}
			idx, delay := producerBlockDelay(candidates, totalValidators)
			downDelay = downDelay + delay
			recents[uint64(h)] = idx
		} else {
			recents[uint64(h)] = proposer
		}
	}
	fmt.Printf("average delay is %v  when there is %d validators and %d is down \n",
		downDelay/time.Duration(downBlocks), totalValidators, downValidators)

	for i := 0; i < downValidators; i++ {
		validators[down[i]] = true
	}

	recoverDelay := time.Duration(0)
	lastseen := downBlocks
	for h := downBlocks + 1; h <= downBlocks+recoverBlocks; h++ {
		if limit := uint64(totalValidators/2 + 1); uint64(h) >= limit {
			delete(recents, uint64(h)-limit)
		}
		proposer := h % totalValidators
		if !isInService(proposer) || isRecentSign(proposer) {
			lastseen = h
			candidates := make(map[int]bool, totalValidators/2)
			for v := range validators {
				if !isRecentSign(v) {
					candidates[v] = true
				}
			}
			if len(candidates) == 0 {
				panic("can not test such case")
			}
			idx, delay := producerBlockDelay(candidates, totalValidators)
			recoverDelay = recoverDelay + delay
			recents[uint64(h)] = idx
		} else {
			recents[uint64(h)] = proposer
		}
	}
	fmt.Printf("total delay is %v after recover when there is %d validators down ever, last seen not proposer at height %d\n",
		recoverDelay, downValidators, lastseen)
}

func producerBlockDelay(candidates map[int]bool, numOfValidators int) (int, time.Duration) {
	minDur := time.Duration(0)
	minIdx := 0
	wiggle := time.Duration(numOfValidators/2+1) * wiggleTime
	for idx := range candidates {
		sleepTime := rand.Int63n(int64(wiggle))
		if int64(minDur) < sleepTime {
			minDur = time.Duration(rand.Int63n(int64(wiggle)))
			minIdx = idx
		}
	}
	return minIdx, minDur
}

func randomAddress() common.Address {
	addrBytes := make([]byte, 20)
	rand.Read(addrBytes)
	return common.BytesToAddress(addrBytes)
}
