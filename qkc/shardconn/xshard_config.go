// Copyright 2026-2027, QuarkChain.

package shardconn

import "time"

// defaultDialTimeout bounds outbound TCP connection establishment.
const defaultDialTimeout = 10 * time.Second

// xshardHandshakeTimeout bounds the inbound PING and outbound PONG waits.
const xshardHandshakeTimeout = 10 * time.Second
