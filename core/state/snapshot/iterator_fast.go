// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package snapshot

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

// fastAccountIterator is a more optimized multi-layer iterator which maintains a
// direct mapping of all iterators leading down to the bottom layer
type fastAccountIterator struct {
	iterators []AccountIterator
	initiated bool
	fail      error
}

// The fast iterator does not query parents as much.
func (dl *diffLayer) newFastAccountIterator() AccountIterator {
	f := &fastAccountIterator{
		iterators: dl.iterators(),
		initiated: false,
	}
	f.Seek(common.Hash{})
	return f
}

// Len returns the number of active iterators
func (fi *fastAccountIterator) Len() int {
	return len(fi.iterators)
}

// Less implements sort.Interface
func (fi *fastAccountIterator) Less(i, j int) bool {
	a := fi.iterators[i].Key()
	b := fi.iterators[j].Key()
	return bytes.Compare(a[:], b[:]) < 0
}

// Swap implements sort.Interface
func (fi *fastAccountIterator) Swap(i, j int) {
	fi.iterators[i], fi.iterators[j] = fi.iterators[j], fi.iterators[i]
}

func (fi *fastAccountIterator) Seek(key common.Hash) {
	// We need to apply this across all iterators
	var seen = make(map[common.Hash]struct{})

	length := len(fi.iterators)
	for i, it := range fi.iterators {
		it.Seek(key)
		for {
			if !it.Next() {
				// To be removed
				// swap it to the last position for now
				fi.iterators[i], fi.iterators[length-1] = fi.iterators[length-1], fi.iterators[i]
				length--
				break
			}
			v := it.Key()
			if _, exist := seen[v]; !exist {
				seen[v] = struct{}{}
				break
			}
		}
	}
	// Now remove those that were placed in the end
	fi.iterators = fi.iterators[:length]
	// The list is now totally unsorted, need to re-sort the entire list
	sort.Sort(fi)
	fi.initiated = false
}

// Next implements the Iterator interface. It returns false if no more elemnts
// can be retrieved (false == exhausted)
func (fi *fastAccountIterator) Next() bool {
	if len(fi.iterators) == 0 {
		return false
	}
	if !fi.initiated {
		// Don't forward first time -- we had to 'Next' once in order to
		// do the sorting already
		fi.initiated = true
		return true
	}
	return fi.innerNext(0)
}

// innerNext handles the next operation internally,
// and should be invoked when we know that two elements in the list may have
// the same value.
// For example, if the list becomes [2,3,5,5,8,9,10], then we should invoke
// innerNext(3), which will call Next on elem 3 (the second '5'). It will continue
// along the list and apply the same operation if needed
func (fi *fastAccountIterator) innerNext(pos int) bool {
	if !fi.iterators[pos].Next() {
		//Exhausted, remove this iterator
		fi.remove(pos)
		if len(fi.iterators) == 0 {
			return false
		}
		return true
	}
	if pos == len(fi.iterators)-1 {
		// Only one iterator left
		return true
	}
	// We next:ed the elem at 'pos'. Now we may have to re-sort that elem
	val, neighbour := fi.iterators[pos].Key(), fi.iterators[pos+1].Key()
	diff := bytes.Compare(val[:], neighbour[:])
	if diff < 0 {
		// It is still in correct place
		return true
	}
	if diff == 0 {
		// It has same value as the neighbour. So still in correct place, but
		// we need to iterate on the neighbour
		fi.innerNext(pos + 1)
		return true
	}
	// At this point, the elem is in the wrong location, but the
	// remaining list is sorted. Find out where to move the elem
	iterationNeeded := false
	index := sort.Search(len(fi.iterators), func(n int) bool {
		if n <= pos {
			// No need to search 'behind' us
			return false
		}
		if n == len(fi.iterators)-1 {
			// Can always place an elem last
			return true
		}
		neighbour := fi.iterators[n+1].Key()
		diff := bytes.Compare(val[:], neighbour[:])
		if diff == 0 {
			// The elem we're placing it next to has the same value,
			// so it's going to need further iteration
			iterationNeeded = true
		}
		return diff < 0
	})
	fi.move(pos, index)
	if iterationNeeded {
		fi.innerNext(index)
	}
	return true
}

// move moves an iterator to another position in the list
func (fi *fastAccountIterator) move(index, newpos int) {
	if newpos > len(fi.iterators)-1 {
		newpos = len(fi.iterators) - 1
	}
	var (
		elem   = fi.iterators[index]
		middle = fi.iterators[index+1 : newpos+1]
		suffix []AccountIterator
	)
	if newpos < len(fi.iterators)-1 {
		suffix = fi.iterators[newpos+1:]
	}
	fi.iterators = append(fi.iterators[:index], middle...)
	fi.iterators = append(fi.iterators, elem)
	fi.iterators = append(fi.iterators, suffix...)
}

// remove drops an iterator from the list
func (fi *fastAccountIterator) remove(index int) {
	fi.iterators = append(fi.iterators[:index], fi.iterators[index+1:]...)
}

// Error returns any failure that occurred during iteration, which might have
// caused a premature iteration exit (e.g. snapshot stack becoming stale).
func (fi *fastAccountIterator) Error() error {
	return fi.fail
}

// Key returns the current key
func (fi *fastAccountIterator) Key() common.Hash {
	return fi.iterators[0].Key()
}

// Value returns the current key
func (fi *fastAccountIterator) Value() []byte {
	panic("todo")
}

// Debug is a convencience helper during testing
func (fi *fastAccountIterator) Debug() {
	for _, it := range fi.iterators {
		fmt.Printf(" %v ", it.Key()[31])
	}
	fmt.Println()
}
