package blockstm

import (
	"fmt"
	"sort"
)

func makeStatusManager(numTasks int) (t taskStatusManager) {
	t.pending = make([]int, numTasks)
	for i := 0; i < numTasks; i++ {
		t.pending[i] = i
	}

	t.dependency = make(map[int]map[int]bool, numTasks)
	t.blockCount = make(map[int]int, numTasks)

	for i := 0; i < numTasks; i++ {
		t.blockCount[i] = -1
	}

	return
}

type taskStatusManager struct {
	pending    []int
	inProgress []int
	complete   []int
	dependency map[int]map[int]bool
	blockCount map[int]int
}

func insertInList(l []int, v int) []int {
	if len(l) == 0 || v > l[len(l)-1] {
		return append(l, v)
	} else {
		x := sort.SearchInts(l, v)
		if x < len(l) && l[x] == v {
			// already in list
			return l
		}
		a := append(l[:x+1], l[x:]...)
		a[x] = v
		return a
	}
}

func (m *taskStatusManager) takeNextPending() int {
	if len(m.pending) == 0 {
		return -1
	}

	x := m.pending[0]
	m.pending = m.pending[1:]
	m.inProgress = insertInList(m.inProgress, x)

	return x
}

func (m *taskStatusManager) peekPendingGE(n int) int {
	x := sort.SearchInts(m.pending, n)
	if x >= len(m.pending) {
		return -1
	}

	return m.pending[x]
}

// Take a pending task whose transaction index is greater than or equal to the given tx index
func (m *taskStatusManager) takePendingGE(n int) int {
	x := sort.SearchInts(m.pending, n)
	if x >= len(m.pending) {
		return -1
	}

	v := m.pending[x]

	if x < len(m.pending)-1 {
		m.pending = append(m.pending[:x], m.pending[x+1:]...)
	} else {
		m.pending = m.pending[:x]
	}

	m.inProgress = insertInList(m.inProgress, v)

	return v
}

func hasNoGap(l []int) bool {
	return l[0]+len(l) == l[len(l)-1]+1
}

func (m taskStatusManager) maxAllComplete() int {
	if len(m.complete) == 0 || m.complete[0] != 0 {
		return -1
	} else if m.complete[len(m.complete)-1] == len(m.complete)-1 {
		return m.complete[len(m.complete)-1]
	} else {
		for i := len(m.complete) - 2; i >= 0; i-- {
			if hasNoGap(m.complete[:i+1]) {
				return m.complete[i]
			}
		}
	}

	return -1
}

func (m *taskStatusManager) pushPending(tx int) {
	if !m.checkComplete(tx) && !m.checkInProgress(tx) {
		m.pending = insertInList(m.pending, tx)
	} else {
		panic(fmt.Errorf("should not happen - clear complete or inProgress before pushing pending"))
	}
}

func removeFromList(l []int, v int, expect bool) []int {
	x := sort.SearchInts(l, v)
	if x == -1 || l[x] != v {
		if expect {
			panic(fmt.Errorf("should not happen - element expected in list"))
		}

		return l
	}

	switch x {
	case 0:
		return l[1:]
	case len(l) - 1:
		return l[:len(l)-1]
	default:
		return append(l[:x], l[x+1:]...)
	}
}

func (m *taskStatusManager) markComplete(tx int) {
	m.inProgress = removeFromList(m.inProgress, tx, true)
	m.complete = insertInList(m.complete, tx)
}

func (m *taskStatusManager) minPending() int {
	if len(m.pending) == 0 {
		return -1
	} else {
		return m.pending[0]
	}
}

func (m *taskStatusManager) countComplete() int {
	return len(m.complete)
}

func (m *taskStatusManager) addDependencies(blocker int, dependent int) bool {
	if blocker < 0 || blocker >= dependent {
		return false
	}

	curBlocker := m.blockCount[dependent]

	if curBlocker > blocker {
		return true
	}

	if m.checkComplete(blocker) {
		// Blocking blocker has already completed
		m.blockCount[dependent] = -1
		return false
	}

	if _, ok := m.dependency[blocker]; !ok {
		m.dependency[blocker] = make(map[int]bool)
	}

	m.dependency[blocker][dependent] = true
	m.blockCount[dependent] = blocker

	return true
}

func (m *taskStatusManager) removeDependency(tx int) {
	if deps, ok := m.dependency[tx]; ok && len(deps) > 0 {
		for k := range deps {
			if m.blockCount[k] == tx {
				m.blockCount[k] = -1
				if !m.checkComplete(k) && !m.checkPending(k) && !m.checkInProgress(k) {
					m.pushPending(k)
				}
			}
		}

		delete(m.dependency, tx)
	}
}

func (m *taskStatusManager) clearInProgress(tx int) {
	m.inProgress = removeFromList(m.inProgress, tx, true)
}

func (m *taskStatusManager) checkInProgress(tx int) bool {
	x := sort.SearchInts(m.inProgress, tx)
	if x < len(m.inProgress) && m.inProgress[x] == tx {
		return true
	}

	return false
}

func (m *taskStatusManager) checkPending(tx int) bool {
	x := sort.SearchInts(m.pending, tx)
	if x < len(m.pending) && m.pending[x] == tx {
		return true
	}

	return false
}

func (m *taskStatusManager) checkComplete(tx int) bool {
	x := sort.SearchInts(m.complete, tx)
	if x < len(m.complete) && m.complete[x] == tx {
		return true
	}

	return false
}

// getRevalidationRange: this range will be all tasks from tx (inclusive) that are not currently in progress up to the
//
//	'all complete' limit
func (m *taskStatusManager) getRevalidationRange(txFrom int) (ret []int) {
	max := m.maxAllComplete() // haven't learned to trust compilers :)
	for x := txFrom; x <= max; x++ {
		if !m.checkInProgress(x) {
			ret = append(ret, x)
		}
	}

	return
}

func (m *taskStatusManager) pushPendingSet(set []int) {
	for _, v := range set {
		if m.checkComplete(v) {
			m.clearComplete(v)
		}

		if !m.checkInProgress(v) {
			m.pushPending(v)
		}
	}
}

func (m *taskStatusManager) clearComplete(tx int) {
	m.complete = removeFromList(m.complete, tx, false)
}

func (m *taskStatusManager) clearPending(tx int) {
	m.pending = removeFromList(m.pending, tx, false)
}
