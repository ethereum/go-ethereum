package portalwire

//go:generate sszgen --path p2p/discover/portalwire/messages.go --exclude-objs BlockHeaderProof,PortalReceipts

// Message codes for the portal protocol.
const (
	PING        byte = 0x00
	PONG        byte = 0x01
	FINDNODES   byte = 0x02
	NODES       byte = 0x03
	FINDCONTENT byte = 0x04
	CONTENT     byte = 0x05
	OFFER       byte = 0x06
	ACCEPT      byte = 0x07
)

// Content selectors for the portal protocol.
const (
	ContentConnIdSelector byte = 0x00
	ContentRawSelector    byte = 0x01
	ContentEnrsSelector   byte = 0x02
)

// Offer request types for the portal protocol.
const (
	OfferRequestDirect   byte = 0x00
	OfferRequestDatabase byte = 0x01
)

const (
	ContentKeysLimit = 64
	// OfferMessageOverhead overhead of content message is a result of 1byte for kind enum, and
	// 4 bytes for offset in ssz serialization
	OfferMessageOverhead = 5

	// PerContentKeyOverhead each key in ContentKeysList has uint32 offset which results in 4 bytes per
	// key overhead when serialized
	PerContentKeyOverhead = 4
)

// Protocol IDs for the portal protocol.
var (
	StateNetwork             = []byte{0x50, 0x0a}
	HistoryNetwork           = []byte{0x50, 0x0b}
	TxGossipNetwork          = []byte{0x50, 0x0c}
	HeaderGossipNetwork      = []byte{0x50, 0x0d}
	CanonicalIndicesNetwork  = []byte{0x50, 0x0e}
	BeaconLightClientNetwork = []byte{0x50, 0x1a}
	UTPNetwork               = []byte{0x75, 0x74, 0x70}
	Rendezvous               = []byte{0x72, 0x65, 0x6e}
)

type ContentKV struct {
	ContentKey []byte
	Content    []byte
}

// Request messages for the portal protocol.
type (
	PingPongCustomData struct {
		Radius []byte `ssz-size:"32"`
	}

	Ping struct {
		EnrSeq        uint64
		CustomPayload []byte `ssz-max:"2048"`
	}

	FindNodes struct {
		Distances [][2]byte `ssz-max:"256,2" ssz-size:"?,2"`
	}

	FindContent struct {
		ContentKey []byte `ssz-max:"2048"`
	}

	Offer struct {
		ContentKeys [][]byte `ssz-max:"64,2048"`
	}
)

// Response messages for the portal protocol.
type (
	Pong struct {
		EnrSeq        uint64
		CustomPayload []byte `ssz-max:"2048"`
	}

	Nodes struct {
		Total uint8
		Enrs  [][]byte `ssz-max:"32,2048"`
	}

	ConnectionId struct {
		Id []byte `ssz-size:"2"`
	}

	Content struct {
		Content []byte `ssz-max:"2048"`
	}

	Enrs struct {
		Enrs [][]byte `ssz-max:"32,2048"`
	}

	Accept struct {
		ConnectionId []byte `ssz-size:"2"`
		ContentKeys  []byte `ssz:"bitlist" ssz-max:"64"`
	}
)
