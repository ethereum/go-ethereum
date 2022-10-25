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
func NewContextWithHeaders(ctx context.Context, src http.Header) context.Context {
	dst, ok := ctx.Value(mdHeaderKey{}).(http.Header)
	if !ok {
		dst = http.Header{}
		ctx = context.WithValue(ctx, mdHeaderKey{}, dst)
	}
	mergeHeaders(dst, src)
	return ctx
}

// headersFromContext is used to extract http.Header from context
func headersFromContext(ctx context.Context) http.Header {
	source, ok := ctx.Value(mdHeaderKey{}).(http.Header)
	if !ok {
		return nil
	}
	return source
}

// mergeHeaders is used to merge src into dst
func mergeHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, val := range values {
			dst.Add(key, val)
		}
	}
}
