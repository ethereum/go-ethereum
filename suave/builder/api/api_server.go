package api

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

// sessionManager is the backend that manages the session state of the builder API.
type sessionManager interface {
	NewSession() (string, error)
	AddTransaction(sessionId string, tx *types.Transaction) (*types.SimulateTransactionResult, error)
}

func NewServer(s sessionManager) *Server {
	api := &Server{
		sessionMngr: s,
	}
	return api
}

type Server struct {
	sessionMngr sessionManager
}

func (s *Server) NewSession(ctx context.Context) (string, error) {
	return s.sessionMngr.NewSession()
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
