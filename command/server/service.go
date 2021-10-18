package server

import (
	"context"

	"github.com/ethereum/go-ethereum/command/server/proto"
)

func (s *Server) Debug(ctx context.Context, req *proto.DebugInput) (*proto.DebugInput, error) {
	return &proto.DebugInput{}, nil
}
