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

func getTestFeedUpdate() *Update {
	return &Update{
		ID:   *getTestID(),
		data: []byte("El que lee mucho y anda mucho, ve mucho y sabe mucho"),
	}
}

func TestUpdateSerializer(t *testing.T) {
	testBinarySerializerRecovery(t, getTestFeedUpdate(), "0x0000000000000000776f726c64206e657773207265706f72742c20657665727920686f7572000000876a8936a7cd0b79ef0735ad0896c1afe278781ce803000000000019456c20717565206c6565206d7563686f207920616e6461206d7563686f2c207665206d7563686f20792073616265206d7563686f")
}

func TestUpdateLengthCheck(t *testing.T) {
	testBinarySerializerLengthCheck(t, getTestFeedUpdate())
	// Test fail if update is too big
	update := getTestFeedUpdate()
	update.data = make([]byte, MaxUpdateDataLength+100)
	serialized := make([]byte, update.binaryLength())
	if err := update.binaryPut(serialized); err == nil {
		t.Fatal("Expected update.binaryPut to fail since update is too big")
	}

	// test fail if data is empty or nil
	update.data = nil
	serialized = make([]byte, update.binaryLength())
	if err := update.binaryPut(serialized); err == nil {
		t.Fatal("Expected update.binaryPut to fail since data is empty")
	}
}
