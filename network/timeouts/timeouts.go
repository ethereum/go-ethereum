package timeouts

import "time"

// FailedPeerSkipDelay is the time we consider a peer to be skipped for a particular request/chunk,
// because this peer failed to deliver it during the SearchTimeout interval
var FailedPeerSkipDelay = 20 * time.Second

// FetcherGlobalTimeout is the max time a node tries to find a chunk for a client, after which it returns a 404
// Basically this is the amount of time a singleflight request for a given chunk lives
var FetcherGlobalTimeout = 10 * time.Second

// SearchTimeout is the max time requests wait for a peer to deliver a chunk, after which another peer is tried
var SearchTimeout = 500 * time.Millisecond

// SyncerClientWaitTimeout is the max time a syncer client waits for a chunk to be delivered during syncing
var SyncerClientWaitTimeout = 20 * time.Second

// Within handleOfferedHashesMsg - how long to wait for a given batch of chunks to be delivered by the peer offering them
var SyncBatchTimeout = 10 * time.Second

// Within SwarmSyncerServer - If at least one chunk is added to the batch and no new chunks
// are added in BatchTimeout period, the batch will be returned.
var BatchTimeout = 2 * time.Second
