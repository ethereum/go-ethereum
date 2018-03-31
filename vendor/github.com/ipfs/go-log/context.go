package log

import (
	"context"
	"errors"
)

type key int

const metadataKey key = 0

// ContextWithLoggable returns a derived context which contains the provided
// Loggable. Any Events logged with the derived context will include the
// provided Loggable.
func ContextWithLoggable(ctx context.Context, l Loggable) context.Context {
	existing, err := MetadataFromContext(ctx)
	if err != nil {
		// context does not contain meta. just set the new metadata
		child := context.WithValue(ctx, metadataKey, Metadata(l.Loggable()))
		return child
	}

	merged := DeepMerge(existing, l.Loggable())
	child := context.WithValue(ctx, metadataKey, merged)
	return child
}

// MetadataFromContext extracts Matadata from a given context's value.
func MetadataFromContext(ctx context.Context) (Metadata, error) {
	value := ctx.Value(metadataKey)
	if value != nil {
		metadata, ok := value.(Metadata)
		if ok {
			return metadata, nil
		}
	}
	return nil, errors.New("context contains no metadata")
}
