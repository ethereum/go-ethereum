// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

// Package cluster defines contracts shared by chain and master business code.
// Backends implement Sender by resolving a destination to a TCP connection;
// they then use that connection's generic command or RPC send path.
package cluster

import (
	"context"

	"github.com/ethereum/go-ethereum/qkc/wire"
)

// Sender is the outbound contract between business code and its backend. On a
// chain backend, wire.RootChainID selects its master connection and an ordinary
// chain ID selects a connection from its chain pool. On a master backend,
// wire.RootChainID is invalid and an ordinary chain ID selects a connection
// from its chain pool. Each call selects one TCP connection; the backend then
// uses that connection's generic non-RPC or RPC send path. Sender does not
// broadcast or gain a method when a new wire message is added.
type Sender interface {
	// Send writes a command to chainID without waiting for a response. An
	// invalid or unresolved destination is an error.
	Send(ctx context.Context, chainID uint32, op byte, msg wire.SSZMessage) error
	// Call sends an RPC to chainID and returns the decoded response type
	// registered for op. The caller checks application-level response status.
	Call(ctx context.Context, chainID uint32, op byte, req wire.SSZMessage) (wire.SSZMessage, error)
}
