package sctx

import "context"

type ContextKey int

const (
	HTTPRequestIDKey ContextKey = iota
	requestHostKey
)

func SetHost(ctx context.Context, domain string) context.Context {
	return context.WithValue(ctx, requestHostKey, domain)
}

func GetHost(ctx context.Context) string {
	v, ok := ctx.Value(requestHostKey).(string)
	if ok {
		return v
	}
	return ""
}
