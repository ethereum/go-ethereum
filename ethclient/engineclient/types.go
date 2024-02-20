package engineclient

import (
	"github.com/ethereum/go-ethereum/beacon/engine"
)

// APIVersion is a custom type for the engine API version.
type APIVersion int

const (
	Phase0 APIVersion = iota
	ParisV1
	ShanghaiV2
	CancunV3
)

// ForkchoiceUpdatedResponse is the response kind received by the
// engine_forkchoiceUpdatedV1 endpoint.
type ForkchoiceUpdatedResponse struct {
	Status          *engine.PayloadStatusV1 `json:"payloadStatus"`
	PayloadId       *engine.PayloadID       `json:"payloadId"`
	ValidationError string                  `json:"validationError"`
}
