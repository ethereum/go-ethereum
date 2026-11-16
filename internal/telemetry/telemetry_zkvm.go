// Copyright 2026 The go-ethereum Authors
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

//go:build zkvm

// A zkVM guest has nobody to report traces to, and linking OpenTelemetry
// costs it: the otel packages pull in net/http and crypto/tls, whose package
// initialisation alone (elliptic-curve constants, tables) runs millions of
// instructions before main. Under the zkvm tag the API is the same and every
// span is a no-op.
package telemetry

import "context"

// Attribute is a key/value annotation of a span.
type Attribute struct {
	Key   string
	Value any
}

func StringAttribute(key, val string) Attribute      { return Attribute{key, val} }
func Int64Attribute(key string, val int64) Attribute { return Attribute{key, val} }
func IntAttribute(key string, val int) Attribute     { return Attribute{key, val} }
func BoolAttribute(key string, val bool) Attribute   { return Attribute{key, val} }

// Span is a tracing span; it records nothing.
type Span struct{}

func (Span) End()                          {}
func (Span) SetAttributes(...Attribute)    {}
func (Span) RecordError(error)             {}
func (Span) AddEvent(string, ...Attribute) {}

// Tracer is whatever a caller holds as its tracer; it is never used, so an
// OpenTelemetry tracer (which rpc and miner keep) is accepted too.
type Tracer = any

// StartSpan returns the context unchanged, a no-op span and an end function
// that does nothing.
func StartSpan(ctx context.Context, spanName string, attributes ...Attribute) (context.Context, Span, func(*error)) {
	return ctx, Span{}, endSpan
}

// StartSpanWithTracer is StartSpan with an explicit tracer.
func StartSpanWithTracer(ctx context.Context, tracer Tracer, name string, attributes ...Attribute) (context.Context, Span, func(*error)) {
	return ctx, Span{}, endSpan
}

// TracerFromContext returns the tracer.
func TracerFromContext(ctx context.Context) Tracer { return nil }

// RPCInfo describes an RPC call for a server span.
type RPCInfo struct {
	System    string
	Service   string
	Method    string
	RequestID string
}

func StartCallServerSpan(ctx context.Context, tracer Tracer, rpc RPCInfo, others ...Attribute) (context.Context, func(*error)) {
	return ctx, endSpan
}

func StartBatchServerSpan(ctx context.Context, tracer Tracer, system string, batchSize int, others ...Attribute) (context.Context, func(*error)) {
	return ctx, endSpan
}

func StartBatchCallSpan(ctx context.Context, tracer Tracer, rpc RPCInfo, others ...Attribute) (context.Context, func(*error)) {
	return ctx, endSpan
}

func endSpan(*error) {}
