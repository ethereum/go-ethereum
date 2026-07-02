// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package catalyst

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
)

// SszRestServer implements the EIP-8161 SSZ-REST Engine API transport.
// It runs alongside the JSON-RPC Engine API and shares the same ConsensusAPI.
type SszRestServer struct {
	api       *ConsensusAPI
	jwtSecret []byte
	addr      string
	port      int
	server    *http.Server
}

// NewSszRestServer creates a new SSZ-REST server.
func NewSszRestServer(api *ConsensusAPI, jwtSecret []byte, addr string, port int) *SszRestServer {
	return &SszRestServer{
		api:       api,
		jwtSecret: jwtSecret,
		addr:      addr,
		port:      port,
	}
}

// Start implements node.Lifecycle. It starts the SSZ-REST HTTP server.
func (s *SszRestServer) Start() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	handler := node.NewJWTHandler(s.jwtSecret, s.panicRecovery(mux))

	listenAddr := fmt.Sprintf("%s:%d", s.addr, s.port)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("[SSZ-REST] failed to listen on %s: %w", listenAddr, err)
	}

	s.server = &http.Server{Handler: handler}
	log.Info("[SSZ-REST] Engine API server started", "addr", listenAddr)

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Error("[SSZ-REST] Server error", "err", err)
		}
	}()

	return nil
}

// Stop implements node.Lifecycle. It stops the SSZ-REST HTTP server.
func (s *SszRestServer) Stop() error {
	if s.server != nil {
		log.Info("[SSZ-REST] Stopping server")
		return s.server.Close()
	}
	return nil
}

// panicRecovery wraps a handler with panic recovery.
func (s *SszRestServer) panicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("[SSZ-REST] panic in handler", "panic", rec, "path", r.URL.Path)
				sszErrorResponse(w, http.StatusInternalServerError, -32603, fmt.Sprintf("internal error: %v", rec))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// sszErrorResponse writes a JSON error response for non-200 status codes per EIP-8161.
func sszErrorResponse(w http.ResponseWriter, code int, jsonRpcCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}{
		Code:    jsonRpcCode,
		Message: message,
	}
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// sszResponse writes a successful SSZ-encoded response.
func sszResponse(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(data) //nolint:errcheck
}

// readBody reads the request body with a size limit.
func readBody(r *http.Request, maxSize int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r.Body, maxSize))
}

// registerRoutes registers all SSZ-REST endpoint routes per EIP-8161.
func (s *SszRestServer) registerRoutes(mux *http.ServeMux) {
	// newPayload versions
	mux.HandleFunc("POST /engine/v1/new_payload", s.handleNewPayloadV1)
	mux.HandleFunc("POST /engine/v2/new_payload", s.handleNewPayloadV2)
	mux.HandleFunc("POST /engine/v3/new_payload", s.handleNewPayloadV3)
	mux.HandleFunc("POST /engine/v4/new_payload", s.handleNewPayloadV4)
	mux.HandleFunc("POST /engine/v5/new_payload", s.handleNewPayloadV5)

	// forkchoiceUpdated versions
	mux.HandleFunc("POST /engine/v1/forkchoice_updated", s.handleForkchoiceUpdatedV1)
	mux.HandleFunc("POST /engine/v2/forkchoice_updated", s.handleForkchoiceUpdatedV2)
	mux.HandleFunc("POST /engine/v3/forkchoice_updated", s.handleForkchoiceUpdatedV3)

	// getPayload versions
	mux.HandleFunc("POST /engine/v1/get_payload", s.handleGetPayloadV1)
	mux.HandleFunc("POST /engine/v2/get_payload", s.handleGetPayloadV2)
	mux.HandleFunc("POST /engine/v3/get_payload", s.handleGetPayloadV3)
	mux.HandleFunc("POST /engine/v4/get_payload", s.handleGetPayloadV4)
	mux.HandleFunc("POST /engine/v5/get_payload", s.handleGetPayloadV5)

	// getBlobs
	mux.HandleFunc("POST /engine/v1/get_blobs", s.handleGetBlobsV1)

	// exchangeCapabilities
	mux.HandleFunc("POST /engine/v1/exchange_capabilities", s.handleExchangeCapabilities)

	// getClientVersion
	mux.HandleFunc("POST /engine/v1/get_client_version", s.handleGetClientVersion)
}

// --- newPayload handlers ---

func (s *SszRestServer) handleNewPayloadV1(w http.ResponseWriter, r *http.Request) {
	s.handleNewPayload(w, r, 1)
}

func (s *SszRestServer) handleNewPayloadV2(w http.ResponseWriter, r *http.Request) {
	s.handleNewPayload(w, r, 2)
}

func (s *SszRestServer) handleNewPayloadV3(w http.ResponseWriter, r *http.Request) {
	s.handleNewPayload(w, r, 3)
}

func (s *SszRestServer) handleNewPayloadV4(w http.ResponseWriter, r *http.Request) {
	s.handleNewPayload(w, r, 4)
}

func (s *SszRestServer) handleNewPayloadV5(w http.ResponseWriter, r *http.Request) {
	s.handleNewPayload(w, r, 5)
}

func (s *SszRestServer) handleNewPayload(w http.ResponseWriter, r *http.Request, version int) {
	log.Info("[SSZ-REST] Received NewPayload", "version", version)

	body, err := readBody(r, 16*1024*1024) // 16 MB max
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}
	if len(body) == 0 {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "empty request body")
		return
	}

	ep, versionedHashes, parentBeaconBlockRoot, executionRequests, err := engine.DecodeNewPayloadRequestSSZ(body, version)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, fmt.Sprintf("SSZ decode error: %v", err))
		return
	}

	ctx := r.Context()
	var result engine.PayloadStatusV1

	switch version {
	case 1:
		result, err = s.api.NewPayloadV1(ctx, *ep)
	case 2:
		result, err = s.api.NewPayloadV2(ctx, *ep)
	case 3:
		result, err = s.api.NewPayloadV3(ctx, *ep, versionedHashes, parentBeaconBlockRoot)
	case 4:
		hexReqs := make([]hexutil.Bytes, len(executionRequests))
		for i, r := range executionRequests {
			hexReqs[i] = hexutil.Bytes(r)
		}
		result, err = s.api.NewPayloadV4(ctx, *ep, versionedHashes, parentBeaconBlockRoot, hexReqs)
	case 5:
		hexReqs := make([]hexutil.Bytes, len(executionRequests))
		for i, r := range executionRequests {
			hexReqs[i] = hexutil.Bytes(r)
		}
		result, err = s.api.NewPayloadV5(ctx, *ep, versionedHashes, parentBeaconBlockRoot, hexReqs)
	default:
		sszErrorResponse(w, http.StatusBadRequest, -32601, fmt.Sprintf("unsupported newPayload version: %d", version))
		return
	}

	if err != nil {
		s.handleEngineError(w, err)
		return
	}

	sszResponse(w, engine.EncodePayloadStatusSSZ(&result))
}

// --- forkchoiceUpdated handlers ---

func (s *SszRestServer) handleForkchoiceUpdatedV1(w http.ResponseWriter, r *http.Request) {
	s.handleForkchoiceUpdated(w, r, 1)
}

func (s *SszRestServer) handleForkchoiceUpdatedV2(w http.ResponseWriter, r *http.Request) {
	s.handleForkchoiceUpdated(w, r, 2)
}

func (s *SszRestServer) handleForkchoiceUpdatedV3(w http.ResponseWriter, r *http.Request) {
	s.handleForkchoiceUpdated(w, r, 3)
}

func (s *SszRestServer) handleForkchoiceUpdated(w http.ResponseWriter, r *http.Request, version int) {
	log.Info("[SSZ-REST] Received ForkchoiceUpdated", "version", version)

	body, err := readBody(r, 1024*1024) // 1 MB max
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}

	const fixedSize = 100 // forkchoice_state(96) + attributes_offset(4)

	if len(body) < 96 {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "request body too short for ForkchoiceState")
		return
	}

	fcs, err := engine.DecodeForkchoiceStateSSZ(body[:96])
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, err.Error())
		return
	}

	var payloadAttributes *engine.PayloadAttributes

	if len(body) >= fixedSize {
		attrOffset := binary.LittleEndian.Uint32(body[96:100])
		if attrOffset <= uint32(len(body)) {
			// List[PayloadAttributesV3SSZ, 1]: for variable-size containers,
			// the list data starts with a 4-byte offset to the first element.
			listData := body[attrOffset:]
			if len(listData) > 4 {
				elemOffset := binary.LittleEndian.Uint32(listData[0:4])
				if elemOffset <= uint32(len(listData)) {
					pa, err := engine.DecodePayloadAttributesSSZ(listData[elemOffset:], version)
					if err != nil {
						sszErrorResponse(w, http.StatusBadRequest, -32602, err.Error())
						return
					}
					payloadAttributes = pa
				}
			}
		}
	}

	ctx := r.Context()
	var resp engine.ForkChoiceResponse

	switch version {
	case 1:
		resp, err = s.api.ForkchoiceUpdatedV1(*fcs, payloadAttributes)
	case 2:
		resp, err = s.api.ForkchoiceUpdatedV2(*fcs, payloadAttributes)
	case 3:
		resp, err = s.api.ForkchoiceUpdatedV3(*fcs, payloadAttributes)
	default:
		sszErrorResponse(w, http.StatusBadRequest, -32601, fmt.Sprintf("unsupported forkchoiceUpdated version: %d", version))
		return
	}
	_ = ctx // context used by api methods internally

	if err != nil {
		s.handleEngineError(w, err)
		return
	}

	sszResponse(w, engine.EncodeForkChoiceResponseSSZ(&resp))
}

// --- getPayload handlers ---

func (s *SszRestServer) handleGetPayloadV1(w http.ResponseWriter, r *http.Request) {
	s.handleGetPayload(w, r, 1)
}

func (s *SszRestServer) handleGetPayloadV2(w http.ResponseWriter, r *http.Request) {
	s.handleGetPayload(w, r, 2)
}

func (s *SszRestServer) handleGetPayloadV3(w http.ResponseWriter, r *http.Request) {
	s.handleGetPayload(w, r, 3)
}

func (s *SszRestServer) handleGetPayloadV4(w http.ResponseWriter, r *http.Request) {
	s.handleGetPayload(w, r, 4)
}

func (s *SszRestServer) handleGetPayloadV5(w http.ResponseWriter, r *http.Request) {
	s.handleGetPayload(w, r, 5)
}

func (s *SszRestServer) handleGetPayload(w http.ResponseWriter, r *http.Request, version int) {
	log.Info("[SSZ-REST] Received GetPayload", "version", version)

	body, err := readBody(r, 64)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}
	if len(body) != 8 {
		sszErrorResponse(w, http.StatusBadRequest, -32602, fmt.Sprintf("expected 8 bytes for payload ID, got %d", len(body)))
		return
	}

	var payloadID engine.PayloadID
	copy(payloadID[:], body)

	switch version {
	case 1:
		result, err := s.api.GetPayloadV1(payloadID)
		if err != nil {
			s.handleEngineError(w, err)
			return
		}
		envelope := &engine.ExecutionPayloadEnvelope{ExecutionPayload: result}
		sszResponse(w, engine.EncodeExecutionPayloadEnvelopeSSZ(envelope, 1))
	case 2:
		result, err := s.api.GetPayloadV2(payloadID)
		if err != nil {
			s.handleEngineError(w, err)
			return
		}
		sszResponse(w, engine.EncodeExecutionPayloadEnvelopeSSZ(result, 2))
	case 3:
		result, err := s.api.GetPayloadV3(payloadID)
		if err != nil {
			s.handleEngineError(w, err)
			return
		}
		sszResponse(w, engine.EncodeExecutionPayloadEnvelopeSSZ(result, 3))
	case 4:
		result, err := s.api.GetPayloadV4(payloadID)
		if err != nil {
			s.handleEngineError(w, err)
			return
		}
		sszResponse(w, engine.EncodeExecutionPayloadEnvelopeSSZ(result, 4))
	case 5:
		result, err := s.api.GetPayloadV5(payloadID)
		if err != nil {
			s.handleEngineError(w, err)
			return
		}
		// V5 uses same payload format as V4 for SSZ encoding
		sszResponse(w, engine.EncodeExecutionPayloadEnvelopeSSZ(result, 4))
	default:
		sszErrorResponse(w, http.StatusBadRequest, -32601, fmt.Sprintf("unsupported getPayload version: %d", version))
	}
}

// --- getBlobs handler ---

func (s *SszRestServer) handleGetBlobsV1(w http.ResponseWriter, r *http.Request) {
	log.Info("[SSZ-REST] Received GetBlobsV1")

	body, err := readBody(r, 1024*1024)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}

	hashes, err := engine.DecodeGetBlobsRequestSSZ(body)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, err.Error())
		return
	}

	result, err := s.api.GetBlobsV1(hashes)
	if err != nil {
		s.handleEngineError(w, err)
		return
	}

	sszResponse(w, encodeGetBlobsV1Response(result))
}

func encodeGetBlobsV1Response(blobs []*engine.BlobAndProofV1) []byte {
	const blobAndProofSize = 131072 + 48

	var count int
	for _, b := range blobs {
		if b != nil {
			count++
		}
	}

	fixedSize := 4 // list_offset
	listSize := count * blobAndProofSize
	buf := make([]byte, fixedSize+listSize)

	binary.LittleEndian.PutUint32(buf[0:4], uint32(fixedSize))

	pos := fixedSize
	for _, b := range blobs {
		if b == nil {
			continue
		}
		copy(buf[pos:pos+131072], b.Blob)
		pos += 131072
		copy(buf[pos:pos+48], b.Proof)
		pos += 48
	}

	return buf
}

// --- exchangeCapabilities handler ---

func (s *SszRestServer) handleExchangeCapabilities(w http.ResponseWriter, r *http.Request) {
	log.Info("[SSZ-REST] Received ExchangeCapabilities")

	body, err := readBody(r, 1024*1024)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}

	capabilities, err := engine.DecodeCapabilitiesSSZ(body)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, err.Error())
		return
	}

	result := s.api.ExchangeCapabilities(capabilities)
	sszResponse(w, engine.EncodeCapabilitiesSSZ(result))
}

// --- getClientVersion handler ---

func (s *SszRestServer) handleGetClientVersion(w http.ResponseWriter, r *http.Request) {
	log.Info("[SSZ-REST] Received GetClientVersion")

	body, err := readBody(r, 1024*1024)
	if err != nil {
		sszErrorResponse(w, http.StatusBadRequest, -32602, "failed to read request body")
		return
	}

	var callerVersion engine.ClientVersionV1
	if len(body) > 0 {
		cv, err := engine.DecodeClientVersionSSZ(body)
		if err != nil {
			sszErrorResponse(w, http.StatusBadRequest, -32602, err.Error())
			return
		}
		callerVersion = *cv
	}

	result := s.api.GetClientVersionV1(callerVersion)
	sszResponse(w, engine.EncodeClientVersionsSSZ(result))
}

// handleEngineError converts engine errors to appropriate HTTP error responses.
func (s *SszRestServer) handleEngineError(w http.ResponseWriter, err error) {
	log.Warn("[SSZ-REST] Engine error", "err", err)
	if engineErr, ok := err.(*engine.EngineAPIError); ok {
		code := engineErr.ErrorCode()
		switch {
		case code == -32602: // InvalidParams
			sszErrorResponse(w, http.StatusBadRequest, code, err.Error())
		case code == -38005: // UnsupportedFork
			sszErrorResponse(w, http.StatusBadRequest, code, err.Error())
		default:
			sszErrorResponse(w, http.StatusInternalServerError, code, err.Error())
		}
		return
	}
	sszErrorResponse(w, http.StatusInternalServerError, -32603, err.Error())
}

// Port returns the configured port for this SSZ-REST server.
func (s *SszRestServer) Port() int {
	return s.port
}
