package rpc

import (
	"context"
	"net/http"
)

type rawMd struct {
	headers http.Header
}

type mdOutgoingKey struct{}

// NewOutgoingContext is used to attach http headers into the context
func NewOutgoingContext(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, mdOutgoingKey{}, rawMd{headers: header})
}

// HeadersFromOutgoingContext is used to extract http headers from the context
func HeadersFromOutgoingContext(ctx context.Context) (http.Header, bool) {
	value := ctx.Value(mdOutgoingKey{})
	if value == nil {
		return nil, false
	}
	headers := value.(rawMd).headers
	if headers == nil {
		return nil, false
	}
	return headers, true
}

// mergeHeadersFromOutgoingContext is used to extract http headers from the context and inject it into the provided headers
func addHeadersFromContext(ctx context.Context, headers http.Header) {
	if kvs, ok := HeadersFromOutgoingContext(ctx); ok {
		for key, values := range kvs {
			headers.Del(key)
			for _, val := range values {
				headers.Add(key, val)
			}
		}
	}
}
