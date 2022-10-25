// Copyright 2022 The go-ethereum Authors
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

package rpc

import (
	"context"
	"net/http"
)

type mdHeaderKey struct{}

// NewContextWithHeaders is used to add the http headers from source into the context.
func NewContextWithHeaders(ctx context.Context, source http.Header) context.Context {
	dest, ok := ctx.Value(mdHeaderKey{}).(http.Header)
	if !ok {
		return context.WithValue(ctx, mdHeaderKey{}, source)
	}
	for key, values := range source {
		dest.Del(key)
		for _, val := range values {
			dest.Add(key, val)
		}
	}
	return context.WithValue(ctx, mdHeaderKey{}, dest)
}

// addHeadersFromContext takes any previously added http headers and adds them
// to the dest http.Header.
func addHeadersFromContext(ctx context.Context, dest http.Header) {
	source, ok := ctx.Value(mdHeaderKey{}).(http.Header)
	if !ok {
		return
	}
	for key, values := range source {
		dest.Del(key)
		for _, val := range values {
			dest.Add(key, val)
		}
	}
}
