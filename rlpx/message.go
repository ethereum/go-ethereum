package rlpx

import (
	"io"
	"time"
)

type RawRLPXMessage struct {
	Code       uint64
	Size       uint32 // Size of the raw payload
	Payload    io.Reader
	ReceivedAt time.Time
}

