package api

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

// SessionManager is the backend that manages the session state of the builder API.
type SessionManager interface {
	NewSession(context.Context) (string, error)
	AddTransaction(sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error)
}

func NewServer(s SessionManager) *Server {
	api := &Server{
		sessionMngr: s,
	}
	return api
}

type Server struct {
	sessionMngr SessionManager
}

func (s *Server) NewSession(ctx context.Context) (string, error) {
	return s.sessionMngr.NewSession(ctx)
}

func (s *Server) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	return s.sessionMngr.AddTransaction(sessionId, tx)
}

type MockServer struct {
}

func (s *MockServer) NewSession(ctx context.Context) (string, error) {
	return "", nil
}

func (s *MockServer) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error) {
	return &types.SimulateTransactionResult{}, nil
}
