package portalwire

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Protocol IDs for the portal protocol.
const (
	StateNetwork             = "0x500a"
	HistoryNetwork           = "0x500b"
	TxGossipNetwork          = "0x500c"
	HeaderGossipNetwork      = "0x500d"
	CanonicalIndicesNetwork  = "0x500e"
	BeaconLightClientNetwork = "0x501a"
	UTPNetwork               = "0x757470"
	Rendezvous               = "0x72656e"
)

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

const (
	ContentConnIdSelector byte = 0x00
	ContentRawSelector    byte = 0x01
	ContentEnrsSelector   byte = 0x02
)

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

func getTalkReqOverheadByLen(protocolIdLen int) int {
	return 16 + // IV size
		55 + // header size
		1 + // talkReq msg id
		3 + // rlp encoding outer list, max length will be encoded in 2 bytes
		9 + // request id (max = 8) + 1 byte from rlp encoding byte string
		protocolIdLen + 1 + // + 1 is necessary due to rlp encoding of byte string
		3 + // rlp encoding response byte string, max length in 2 bytes
		16 // HMAC
}

func getTalkReqOverhead(protocolId string) int {
	protocolIdBytes, _ := hexutil.Decode(protocolId)
	return getTalkReqOverheadByLen(len(protocolIdBytes))
}
