package bor

import (
	"github.com/ethereum/go-ethereum/consensus/bor/clerk"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/checkpoint"
	"github.com/ethereum/go-ethereum/consensus/bor/heimdall/span"
)

//go:generate mockgen -destination=../../tests/bor/mocks/IHeimdallClient.go -package=mocks . IHeimdallClient
type IHeimdallClient interface {
	StateSyncEvents(fromID uint64, to int64) ([]*clerk.EventRecordWithTime, error)
	Span(spanID uint64) (*span.HeimdallSpan, error)
	FetchLatestCheckpoint() (*checkpoint.Checkpoint, error)
	Close()
}
