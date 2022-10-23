package rpc

import (
	"context"
	"net/http"
)

type mdHeaderKey struct{}

// NewContextWithHeaders is used to attach http headers into the context
func NewContextWithHeaders(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, mdHeaderKey{}, header)
}

// HeadersFromContext is used to extract http headers from the context
func HeadersFromContext(ctx context.Context) http.Header {
	value := ctx.Value(mdHeaderKey{})
	if value == nil {
		return nil
	}
	return value.(http.Header)
}

// addHeadersFromContext is used to extract http headers from the context and inject it into the provided headers
func addHeadersFromContext(ctx context.Context, headers http.Header) {
	if kvs := HeadersFromContext(ctx); kvs != nil {
		for key, values := range kvs {
			headers.Del(key)
			for _, val := range values {
				headers.Add(key, val)
			}
		}
	}
}
