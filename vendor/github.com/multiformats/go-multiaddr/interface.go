package multiaddr

/*
Multiaddr is a cross-protocol, cross-platform format for representing
internet addresses. It emphasizes explicitness and self-description.
Learn more here: https://github.com/multiformats/multiaddr

Multiaddrs have both a binary and string representation.

    import ma "github.com/multiformats/go-multiaddr"

    addr, err := ma.NewMultiaddr("/ip4/1.2.3.4/tcp/80")
    // err non-nil when parsing failed.

*/
type Multiaddr interface {
	// Equal returns whether two Multiaddrs are exactly equal
	Equal(Multiaddr) bool

	// Bytes returns the []byte representation of this Multiaddr
	Bytes() []byte

	// String returns the string representation of this Multiaddr
	// (may panic if internal state is corrupted)
	String() string

	// Protocols returns the list of Protocols this Multiaddr includes
	// will panic if protocol code incorrect (and bytes accessed incorrectly)
	Protocols() []Protocol

	// Encapsulate wraps this Multiaddr around another. For example:
	//
	//      /ip4/1.2.3.4 encapsulate /tcp/80 = /ip4/1.2.3.4/tcp/80
	//
	Encapsulate(Multiaddr) Multiaddr

	// Decapsultate removes a Multiaddr wrapping. For example:
	//
	//      /ip4/1.2.3.4/tcp/80 decapsulate /ip4/1.2.3.4 = /tcp/80
	//
	Decapsulate(Multiaddr) Multiaddr

	// ValueForProtocol returns the value (if any) following the specified protocol
	ValueForProtocol(code int) (string, error)
}
