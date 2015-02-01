package bzz

/*
DHT implements the chunk store that directly communicates with the bzz protocol on the one hand and the kademlia node table on the other.
It accumulates requests from peers, keeping a request pool and does forwarding for incoming  requests and handles expiry/timeout.
*/

// it implements the ChunkStore interface
type DHTStore struct {
}
