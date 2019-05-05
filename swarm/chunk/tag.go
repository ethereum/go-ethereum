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

package chunk

import (
	"encoding/binary"
	"errors"
	"sync/atomic"
	"time"
)

var (
	errExists      = errors.New("already exists")
	errNA          = errors.New("not available yet")
	errNoETA       = errors.New("unable to calculate ETA")
	errTagNotFound = errors.New("tag not found")
)

// State is the enum type for chunk states
type State = uint32

const (
	StateSplit  State = iota // chunk has been processed by filehasher/swarm safe call
	StateStored              // chunk stored locally
	StateSeen                // chunk previously seen
	StateSent                // chunk sent to neighbourhood
	StateSynced              // proof is received; chunk removed from sync db; chunk is available everywhere
)

// Tag represents info on the status of new chunks
type Tag struct {
	Uid       uint32    // a unique identifier for this tag
	Name      string    // a name tag for this tag
	Address   Address   // the associated swarm hash for this tag
	total     int64     // total chunks belonging to a tag
	split     int64     // number of chunks already processed by splitter for hashing
	seen      int64     // number of chunks already seen
	stored    int64     // number of chunks already stored locally
	sent      int64     // number of chunks sent for push syncing
	synced    int64     // number of chunks synced with proof
	startedAt time.Time // tag started to calculate ETA
}

// New creates a new tag, stores it by the name and returns it
// it returns an error if the tag with this name already exists
func NewTag(uid uint32, s string, total int64) *Tag {
	t := &Tag{
		Uid:       uid,
		Name:      s,
		startedAt: time.Now(),
		total:     total,
	}
	return t
}

// Inc increments the count for a state
func (t *Tag) Inc(state State) {
	var v *int64
	switch state {
	case StateSplit:
		v = &t.split
	case StateStored:
		v = &t.stored
	case StateSeen:
		v = &t.seen
	case StateSent:
		v = &t.sent
	case StateSynced:
		v = &t.synced
	}
	atomic.AddInt64(v, 1)
}

// Get returns the count for a state on a tag
func (t *Tag) Get(state State) int64 {
	var v *int64
	switch state {
	case StateSplit:
		v = &t.split
	case StateStored:
		v = &t.stored
	case StateSeen:
		v = &t.seen
	case StateSent:
		v = &t.sent
	case StateSynced:
		v = &t.synced
	}
	return atomic.LoadInt64(v)
}

// GetTotal returns the total count
func (t *Tag) Total() int64 {
	return atomic.LoadInt64(&t.total)
}

// DoneSplit sets total count to SPLIT count and sets the associated swarm hash for this tag
// is meant to be called when splitter finishes for input streams of unknown size
func (t *Tag) DoneSplit(address Address) int64 {
	total := atomic.LoadInt64(&t.split)
	atomic.StoreInt64(&t.total, total)
	t.Address = address
	return total
}

// Status returns the value of state and the total count
func (t *Tag) Status(state State) (int64, int64, error) {
	count, seen, total := t.Get(state), atomic.LoadInt64(&t.seen), atomic.LoadInt64(&t.total)
	if total == 0 {
		return count, total, errNA
	}
	switch state {
	case StateSplit, StateStored, StateSeen:
		return count, total, nil
	case StateSent, StateSynced:
		stored := atomic.LoadInt64(&t.stored)
		if stored < total {
			return count, total - seen, errNA
		}
		return count, total - seen, nil
	}
	return count, total, errNA
}

// ETA returns the time of completion estimated based on time passed and rate of completion
func (t *Tag) ETA(state State) (time.Time, error) {
	cnt, total, err := t.Status(state)
	if err != nil {
		return time.Time{}, err
	}
	if cnt == 0 || total == 0 {
		return time.Time{}, errNoETA
	}
	diff := time.Since(t.startedAt)
	dur := time.Duration(total) * diff / time.Duration(cnt)
	return t.startedAt.Add(dur), nil
}

// MarshalBinary marshals the tag into a byte slice
func (tag *Tag) MarshalBinary() (data []byte, err error) {
	buffer := make([]byte, 4)
	binary.BigEndian.PutUint32(buffer, tag.Uid)
	encodeInt64Append(&buffer, tag.total)
	encodeInt64Append(&buffer, tag.split)
	encodeInt64Append(&buffer, tag.seen)
	encodeInt64Append(&buffer, tag.stored)
	encodeInt64Append(&buffer, tag.sent)
	encodeInt64Append(&buffer, tag.synced)

	intBuffer := make([]byte, 8)

	n := binary.PutVarint(intBuffer, tag.startedAt.Unix())
	buffer = append(buffer, intBuffer[:n]...)

	n = binary.PutVarint(intBuffer, int64(len(tag.Address)))
	buffer = append(buffer, intBuffer[:n]...)

	buffer = append(buffer, tag.Address[:]...)

	buffer = append(buffer, []byte(tag.Name)...)

	return buffer, nil
}

// UnmarshalBinary unmarshals a byte slice into a tag
func (tag *Tag) UnmarshalBinary(buffer []byte) error {
	if len(buffer) < 13 {
		return errors.New("buffer too short")
	}
	tag.Uid = binary.BigEndian.Uint32(buffer)
	buffer = buffer[4:]

	tag.total = decodeInt64Splice(&buffer)
	tag.split = decodeInt64Splice(&buffer)
	tag.seen = decodeInt64Splice(&buffer)
	tag.stored = decodeInt64Splice(&buffer)
	tag.sent = decodeInt64Splice(&buffer)
	tag.synced = decodeInt64Splice(&buffer)

	t, n := binary.Varint(buffer)
	tag.startedAt = time.Unix(t, 0)
	buffer = buffer[n:]

	t, n = binary.Varint(buffer)
	buffer = buffer[n:]
	if t > 0 {
		tag.Address = buffer[:t]
	}
	tag.Name = string(buffer[t:])

	return nil
}

func encodeInt64Append(buffer *[]byte, val int64) {
	intBuffer := make([]byte, 8)
	n := binary.PutVarint(intBuffer, val)
	*buffer = append(*buffer, intBuffer[:n]...)
}

func decodeInt64Splice(buffer *[]byte) int64 {
	val, n := binary.Varint((*buffer))
	*buffer = (*buffer)[n:]
	return val
}
