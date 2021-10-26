package server

import (
	"context"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/command/server/pprof"
	"github.com/ethereum/go-ethereum/command/server/proto"
)

func (s *Server) Pprof(ctx context.Context, req *proto.PprofRequest) (*proto.PprofResponse, error) {
	var payload []byte
	var headers map[string]string
	var err error

	switch req.Type {
	case proto.PprofRequest_CPU:
		payload, headers, err = pprof.CPUProfile(ctx, int(req.Seconds))
	case proto.PprofRequest_TRACE:
		payload, headers, err = pprof.Trace(ctx, int(req.Seconds))
	case proto.PprofRequest_LOOKUP:
		payload, headers, err = pprof.Profile(req.Profile, 0, 0)
	}
	if err != nil {
		return nil, err
	}

	resp := &proto.PprofResponse{
		Payload: hex.EncodeToString(payload),
		Headers: headers,
	}
	return resp, nil
}
