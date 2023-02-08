package common

import (
	"context"

	unique "github.com/ethereum/go-ethereum/common/set"
)

type key struct{}

var (
	labelsKey key
)

func WithLabels(ctx context.Context, labels ...string) context.Context {
	if len(labels) == 0 {
		return ctx
	}

	labels = append(labels, Labels(ctx)...)

	return context.WithValue(ctx, labelsKey, unique.Deduplicate(labels))
}

func Labels(ctx context.Context) []string {
	labels, ok := ctx.Value(labelsKey).([]string)
	if !ok {
		return nil
	}

	return labels
}
