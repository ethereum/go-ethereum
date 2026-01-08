// Copyright 2025 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/internal/telemetry/config"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/urfave/cli/v2"
)

// telemetryService wraps the telemetry Provider to implement node.Lifecycle.
type telemetryService struct {
	provider *config.Provider
}

// Start implements node.Lifecycle.
func (t *telemetryService) Start() error {
	return nil // Provider is already started during setup
}

// Stop implements node.Lifecycle.
func (t *telemetryService) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.provider.Shutdown(ctx); err != nil {
		log.Error("Failed to stop telemetry service", "err", err)
		return err
	}
	log.Info("Telemetry stopped")
	return nil
}

// SetupTelemetry initializes OpenTelemetry tracing based on CLI flags.
func SetupTelemetry(ctx *cli.Context) (*telemetryService, error) {
	if !ctx.Bool(RPCTracingFlag.Name) {
		return nil, nil
	}
	endpoint := ctx.String(RPCTracingEndpointFlag.Name)
	if endpoint == "" {
		return nil, nil
	}
	setupCtx := ctx.Context
	if setupCtx == nil {
		setupCtx = context.Background()
	}

	// Configure OpenTelemetry tracing
	handle, err := config.Setup(
		setupCtx,
		endpoint,
	)
	if err != nil {
		return nil, err
	}
	log.Info("Telemetry enabled", "endpoint", endpoint)
	return &telemetryService{provider: handle}, nil
}

// RegisterTelemetryService registers the telemetryService with the node.
func RegisterTelemetryService(service *telemetryService, stack *node.Node) {
	stack.RegisterLifecycle(service)
}
