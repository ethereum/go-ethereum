package api

import (
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

var _ API = (*Server)(nil)

// SessionManager is the backend that manages the session state of the builder API.
type SessionManager interface {
	NewSession(context.Context, *BuildBlockArgs) (string, error)
	AddTransaction(sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error)
	BuildBlock(sessionId string) error
	Bid(sessionId string, blsPubKey string) error
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

func (s *Server) NewSession(ctx context.Context, args *BuildBlockArgs) (string, error) {
	return s.sessionMngr.NewSession(ctx, args)
}

func (s *Server) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error) {
	return s.sessionMngr.AddTransaction(sessionId, tx)
}

func (s *Server) BuildBlock(ctx context.Context, sessionId string) error {
	return s.sessionMngr.BuildBlock(sessionId)
}

func (s *Server) Bid(ctx context.Context, sessioId string, blsPubKey string) error {
	return nil
}

// TODO: Remove
type MockServer struct {
}

func (s *MockServer) NewSession(ctx context.Context, args *BuildBlockArgs) (string, error) {
	return "", nil
}

func (s *MockServer) AddTransaction(ctx context.Context, sessionId string, tx *types.Transaction) (*SimulateTransactionResult, error) {
	return &SimulateTransactionResult{}, nil
}

func (s *MockServer) BuildBlock(ctx context.Context) error {
	return nil
}
