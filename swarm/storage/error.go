package storage

import (
	"errors"
)

const (
	ErrInit = iota
	ErrNotFound
	ErrIO
	ErrUnauthorized
	ErrInvalidValue
	ErrDataOverflow
	ErrNothingToReturn
	ErrInvalidSignature
	ErrNotSynced
	ErrPeriodDepth
	ErrCnt
)

var (
	ErrChunkNotFound    = errors.New("chunk not found")
	ErrFetching         = errors.New("chunk still fetching")
	ErrChunkInvalid     = errors.New("invalid chunk")
	ErrChunkForward     = errors.New("cannot forward")
	ErrChunkUnavailable = errors.New("chunk unavailable")
	ErrChunkTimeout     = errors.New("timeout")
)
