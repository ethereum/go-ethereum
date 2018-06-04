// Copyright 2017 The go-interpreter Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package stack implements a growable uint64 stack
package stack

type Stack struct {
	slice []uint64
}

func (s *Stack) Push(b uint64) {
	s.slice = append(s.slice, b)
}

func (s *Stack) Pop() uint64 {
	v := s.Top()
	s.slice = s.slice[:len(s.slice)-1]
	return v
}

func (s *Stack) SetTop(v uint64) {
	s.slice[len(s.slice)-1] = v
}

func (s *Stack) Top() uint64 {
	return s.slice[len(s.slice)-1]
}

func (s *Stack) Get(i int) uint64 {
	return s.slice[i]
}

func (s *Stack) Set(i int, v uint64) {
	s.slice[i] = v
}

func (s *Stack) Len() int {
	return len(s.slice)
}
