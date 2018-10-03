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
	"testing"
)

func getTestQuery() *Query {
	id := getTestID()
	return &Query{
		TimeLimit: 5000,
		Feed:      id.Feed,
		Hint:      id.Epoch,
	}
}

func TestQueryValues(t *testing.T) {
	var expected = KV{"hint.level": "25", "hint.time": "1000", "time": "5000", "topic": "0x776f726c64206e657773207265706f72742c20657665727920686f7572000000", "user": "0x876A8936A7Cd0b79Ef0735AD0896c1AFe278781c"}

	query := getTestQuery()
	testValueSerializer(t, query, expected)

}
