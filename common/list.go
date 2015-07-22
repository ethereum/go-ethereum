// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package common

import (
	"encoding/json"
	"reflect"
	"sync"
)

// The list type is an anonymous slice handler which can be used
// for containing any slice type to use in an environment which
// does not support slice types (e.g., JavaScript, QML)
type List struct {
	mut    sync.Mutex
	val    interface{}
	list   reflect.Value
	Length int
}

// Initialise a new list. Panics if non-slice type is given.
func NewList(t interface{}) *List {
	list := reflect.ValueOf(t)
	if list.Kind() != reflect.Slice {
		panic("list container initialized with a non-slice type")
	}

	return &List{sync.Mutex{}, t, list, list.Len()}
}

func EmptyList() *List {
	return NewList([]interface{}{})
}

// Get N element from the embedded slice. Returns nil if OOB.
func (self *List) Get(i int) interface{} {
	if self.list.Len() > i {
		self.mut.Lock()
		defer self.mut.Unlock()

		i := self.list.Index(i).Interface()

		return i
	}

	return nil
}

func (self *List) GetAsJson(i int) interface{} {
	e := self.Get(i)

	r, _ := json.Marshal(e)

	return string(r)
}

// Appends value at the end of the slice. Panics when incompatible value
// is given.
func (self *List) Append(v interface{}) {
	self.mut.Lock()
	defer self.mut.Unlock()

	self.list = reflect.Append(self.list, reflect.ValueOf(v))
	self.Length = self.list.Len()
}

// Returns the underlying slice as interface.
func (self *List) Interface() interface{} {
	return self.list.Interface()
}

// For JavaScript <3
func (self *List) ToJSON() string {
	// make(T, 0) != nil
	list := make([]interface{}, 0)
	for i := 0; i < self.Length; i++ {
		list = append(list, self.Get(i))
	}

	data, _ := json.Marshal(list)

	return string(data)
}
