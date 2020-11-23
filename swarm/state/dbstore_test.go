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

package state

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

var ErrInvalidArraySize = errors.New("invalid byte array size")
var ErrInvalidValuePersisted = errors.New("invalid value was persisted to the db")

type SerializingType struct {
	key   string
	value string
}

func (st *SerializingType) MarshalBinary() (data []byte, err error) {
	d := []byte(strings.Join([]string{st.key, st.value}, ";"))

	return d, nil
}

func (st *SerializingType) UnmarshalBinary(data []byte) (err error) {
	d := bytes.Split(data, []byte(";"))
	l := len(d)
	if l == 0 {
		return ErrInvalidArraySize
	}
	if l == 2 {
		keyLen := len(d[0])
		st.key = string(d[0][:keyLen])

		valLen := len(d[1])
		st.value = string(d[1][:valLen])
	}

	return nil
}

// TestDBStore tests basic functionality of DBStore.
func TestDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "db_store_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := NewDBStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	testStore(t, store)

	store.Close()

	persistedStore, err := NewDBStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer persistedStore.Close()

	testPersistedStore(t, persistedStore)
}

func testStore(t *testing.T, store Store) {
	ser := &SerializingType{key: "key1", value: "value1"}
	jsonify := []string{"a", "b", "c"}

	err := store.Put(ser.key, ser)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Put("key2", jsonify)
	if err != nil {
		t.Fatal(err)
	}

}

func testPersistedStore(t *testing.T, store Store) {
	ser := &SerializingType{}

	err := store.Get("key1", ser)
	if err != nil {
		t.Fatal(err)
	}

	if ser.key != "key1" || ser.value != "value1" {
		t.Fatal(ErrInvalidValuePersisted)
	}

	as := []string{}
	err = store.Get("key2", &as)

	if len(as) != 3 {
		t.Fatalf("serialized array did not match expectation")
	}
	if as[0] != "a" || as[1] != "b" || as[2] != "c" {
		t.Fatalf("elements serialized did not match expected values")
	}
}
