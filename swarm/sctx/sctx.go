package sctx

import "context"

type (
	HTTPRequestIDKey struct{}
	requestHostKey   struct{}
	tagKey           struct{}
)

// SetHost sets the http request host in the context
func SetHost(ctx context.Context, domain string) context.Context {
	return context.WithValue(ctx, requestHostKey{}, domain)
}

// GetHost gets the request host from the context
func GetHost(ctx context.Context) string {
	v, ok := ctx.Value(requestHostKey{}).(string)
	if ok {
		return v
	}
	return ""
}

// SetTag sets the tag unique identifier in the context
func SetTag(ctx context.Context, tagId uint32) context.Context {
	return context.WithValue(ctx, tagKey{}, tagId)
}

// GetTag gets the tag unique identifier from the context
func GetTag(ctx context.Context) uint32 {
	v, ok := ctx.Value(tagKey{}).(uint32)
	if ok {
		return v
	}
	return 0
}
