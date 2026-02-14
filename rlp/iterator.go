// Copyright 2020 The go-ethereum Authors
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

package rlp

// Iterator is an iterator over the elements of an encoded container.
type Iterator struct {
	data   []byte
	next   []byte
	offset int
	err    error
}

// NewListIterator creates an iterator for the (list) represented by data.
func NewListIterator(data RawValue) (Iterator, error) {
	k, t, c, err := readKind(data)
	if err != nil {
		return Iterator{}, err
	}
	if k != List {
		return Iterator{}, ErrExpectedList
	}
	it := newIterator(data[t:t+c], int(t))
	return it, nil
}

func newIterator(data []byte, initialOffset int) Iterator {
	return Iterator{data: data, offset: initialOffset}
}

// Next forwards the iterator one step.
// Returns true if there is a next item or an error occurred on this step (check Err()).
// On parse error, the iterator is marked finished and subsequent calls return false.
func (it *Iterator) Next() bool {
	if len(it.data) == 0 {
		return false
	}
	_, t, c, err := readKind(it.data)
	if err != nil {
		it.next = nil
		it.err = err
		// Mark iteration as finished to avoid potential infinite loops on subsequent Next calls.
		it.data = nil
		return true
	}
	length := t + c
	it.next = it.data[:length]
	it.data = it.data[length:]
	it.offset += int(length)
	it.err = nil
	return true
}

// Value returns the current value.
func (it *Iterator) Value() []byte {
	return it.next
}

// Count returns the remaining number of items.
// Note this is O(n) and the result may be incorrect if the list data is invalid.
// The returned count is always an upper bound on the remaining items
// that will be visited by the iterator.
func (it *Iterator) Count() int {
	count, _ := CountValues(it.data)
	return count
}

// Offset returns the offset of the current value into the list data.
func (it *Iterator) Offset() int {
	return it.offset - len(it.next)
}

// Err returns the error that caused Next to return false, if any.
func (it *Iterator) Err() error {
	return it.err
}
