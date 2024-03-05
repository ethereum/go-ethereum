// Copyright 2024 The go-ethereum Authors
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

package lightclient

import (
	"context"
	"sync"
)

type valueAndError[V any] struct {
	value V
	err   error
}

type objectRequest[V any] struct {
	pending  map[chan valueAndError[V]]struct{}
	cancelFn func()
}

type requestMap[K comparable, V any] struct {
	lock     sync.Mutex
	requests map[K]*objectRequest[V]
}

func newRequestMap[K comparable, V any]() *requestMap[K, V] {
	return &requestMap[K, V]{
		requests: make(map[K]*objectRequest[V]),
	}
}

func (r *requestMap[K, V]) add(key K) (chan valueAndError[V], bool) {
	r.lock.Lock()
	defer r.lock.Unlock()

	ch := make(chan valueAndError[V], 1)
	req, ok := r.requests[key]
	if !ok {
		req = &objectRequest[V]{pending: make(map[chan valueAndError[V]]struct{})}
	}
	req.pending[ch] = struct{}{}
	return ch, !ok
}

func (r *requestMap[K, V]) has(key K) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	_, ok := r.requests[key]
	return ok
}

func (r *requestMap[K, V]) requestContext(key K) context.Context {
	r.lock.Lock()
	defer r.lock.Unlock()

	ctx, cancelFn := context.WithCancel(context.Background())
	if req, ok := r.requests[key]; ok {
		req.cancelFn = cancelFn
	} else {
		cancelFn()
	}
	return ctx
}

func (r *requestMap[K, V]) deliver(key K, value V, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.requests[key]
	if !ok {
		return
	}
	for ch := range req.pending {
		ch <- valueAndError[V]{value: value, err: err}
	}
	delete(r.requests, key)
	if req.cancelFn != nil {
		req.cancelFn()
	}
}

func (r *requestMap[K, V]) remove(key K, ch chan valueAndError[V]) {
	r.lock.Lock()
	defer r.lock.Unlock()

	req, ok := r.requests[key]
	if !ok {
		return
	}
	delete(req.pending, ch)
	if len(req.pending) == 0 {
		delete(r.requests, key)
		if req.cancelFn != nil {
			req.cancelFn()
		}
	}
}

func (r *requestMap[K, V]) waitForValue(ctx context.Context, key K, ch chan valueAndError[V]) (V, error) {
	var empty V
	select {
	case v := <-ch:
		return v.value, v.err
	case <-ctx.Done():
		r.remove(key, ch)
		return empty, ctx.Err()
	}
}
