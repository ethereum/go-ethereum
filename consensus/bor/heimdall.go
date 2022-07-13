package bor

import (
	"context"

	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
)

//go:generate mockgen -destination=../../tests/bor/mocks/IHeimdallClient.go -package=mocks . IHeimdallClient
type IHeimdallClient interface {
	StateSyncEvents(ctx context.Context, fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error)
	Span(ctx context.Context, spanID uint64) (*span.HeimdallSpan, error)
	FetchLatestCheckpoint(ctx context.Context) (*checkpoint.Checkpoint, error)
	Close()
}
