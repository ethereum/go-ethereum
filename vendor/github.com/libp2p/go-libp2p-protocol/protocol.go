package protocol

// ID is an identifier used to write protocol headers in streams.
type ID string

// These are reserved protocol.IDs.
const (
	TestingID ID = "/p2p/_testing"
)
