// Copyright 2026-2027, QuarkChain.

package shardconn

import "time"

// defaultDialTimeout bounds outbound TCP connection establishment.
const defaultDialTimeout = 10 * time.Second

// xshardHandshakeTimeout bounds the wait for the initial PING from an
// inbound peer.
const xshardHandshakeTimeout = 10 * time.Second
