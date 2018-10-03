// Copyright 2018 The go-ethereum Authors
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

package feed

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/swarm/storage/feed/lookup"
)

// Query is used to specify constraints when performing an update lookup
// TimeLimit indicates an upper bound for the search. Set to 0 for "now"
type Query struct {
	Feed
	Hint      lookup.Epoch
	TimeLimit uint64
}

// FromValues deserializes this instance from a string key-value store
// useful to parse query strings
func (q *Query) FromValues(values Values) error {
	time, _ := strconv.ParseUint(values.Get("time"), 10, 64)
	q.TimeLimit = time

	level, _ := strconv.ParseUint(values.Get("hint.level"), 10, 32)
	q.Hint.Level = uint8(level)
	q.Hint.Time, _ = strconv.ParseUint(values.Get("hint.time"), 10, 64)
	if q.Feed.User == (common.Address{}) {
		return q.Feed.FromValues(values)
	}
	return nil
}

// AppendValues serializes this structure into the provided string key-value store
// useful to build query strings
func (q *Query) AppendValues(values Values) {
	if q.TimeLimit != 0 {
		values.Set("time", fmt.Sprintf("%d", q.TimeLimit))
	}
	if q.Hint.Level != 0 {
		values.Set("hint.level", fmt.Sprintf("%d", q.Hint.Level))
	}
	if q.Hint.Time != 0 {
		values.Set("hint.time", fmt.Sprintf("%d", q.Hint.Time))
	}
	q.Feed.AppendValues(values)
}

// NewQuery constructs an Query structure to find updates on or before `time`
// if time == 0, the latest update will be looked up
func NewQuery(feed *Feed, time uint64, hint lookup.Epoch) *Query {
	return &Query{
		TimeLimit: time,
		Feed:      *feed,
		Hint:      hint,
	}
}

// NewQueryLatest generates lookup parameters that look for the latest update to a feed
func NewQueryLatest(feed *Feed, hint lookup.Epoch) *Query {
	return NewQuery(feed, 0, hint)
}
