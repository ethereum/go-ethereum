package portalwire

import (
	ssz "github.com/ferranbt/fastssz"
)

// note: We changed the generated file since fastssz issues which can't be passed by the CI, so we commented the go:generate line
///go:generate sszgen --path types.go --exclude-objs Content,Enrs,ContentKV

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

const (
	ContentKeysLimit = 64
	// OfferMessageOverhead overhead of content message is a result of 1byte for kind enum, and
	// 4 bytes for offset in ssz serialization
	OfferMessageOverhead = 5

	// PerContentKeyOverhead each key in ContentKeysList has uint32 offset which results in 4 bytes per
	// key overhead when serialized
	PerContentKeyOverhead = 4
)

type ProtocolId []byte

var (
	State             ProtocolId = []byte{0x50, 0x0A}
	History           ProtocolId = []byte{0x50, 0x0B}
	Beacon            ProtocolId = []byte{0x50, 0x0C}
	CanonicalIndices  ProtocolId = []byte{0x50, 0x0D}
	VerkleState       ProtocolId = []byte{0x50, 0x0E}
	TransactionGossip ProtocolId = []byte{0x50, 0x0F}
	Utp               ProtocolId = []byte{0x75, 0x74, 0x70}
)

var protocolName = map[string]string{
	string(State):             "state",
	string(History):           "history",
	string(Beacon):            "beacon",
	string(CanonicalIndices):  "canonical indices",
	string(VerkleState):       "verkle state",
	string(TransactionGossip): "transaction gossip",
}

func (p ProtocolId) Name() string {
	return protocolName[string(p)]
}

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

// MarshalSSZ ssz marshals the Content object
func (c *Content) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(c)
}

// MarshalSSZTo ssz marshals the Content object to a target array
func (c *Content) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf

	// Field (0) 'Content'
	if size := len(c.Content); size > 2048 {
		err = ssz.ErrBytesLengthFn("Content.Content", size, 2048)
		return
	}
	dst = append(dst, c.Content...)

	return
}

// UnmarshalSSZ ssz unmarshals the Content object
func (c *Content) UnmarshalSSZ(buf []byte) error {
	var err error
	tail := buf

	// Field (0) 'Content'
	{
		buf = tail[:]
		if len(buf) > 2048 {
			return ssz.ErrBytesLength
		}
		if cap(c.Content) == 0 {
			c.Content = make([]byte, 0, len(buf))
		}
		c.Content = append(c.Content, buf...)
	}
	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the Content object
func (c *Content) SizeSSZ() (size int) {
	// Field (0) 'Content'
	return len(c.Content)
}

// HashTreeRoot ssz hashes the Content object
func (c *Content) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(c)
}

// HashTreeRootWith ssz hashes the Content object with a hasher
func (c *Content) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Content'
	{
		elemIndx := hh.Index()
		byteLen := uint64(len(c.Content))
		if byteLen > 2048 {
			err = ssz.ErrIncorrectListSize
			return
		}
		hh.Append(c.Content)
		hh.MerkleizeWithMixin(elemIndx, byteLen, (2048+31)/32)
	}

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the Content object
func (c *Content) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(c)
}

// MarshalSSZ ssz marshals the Enrs object
func (e *Enrs) MarshalSSZ() ([]byte, error) {
	return ssz.MarshalSSZ(e)
}

// MarshalSSZTo ssz marshals the Enrs object to a target array
func (e *Enrs) MarshalSSZTo(buf []byte) (dst []byte, err error) {
	dst = buf
	offset := int(0)

	// Field (0) 'Enrs'
	if size := len(e.Enrs); size > 32 {
		err = ssz.ErrListTooBigFn("Enrs.Enrs", size, 32)
		return
	}
	{
		offset = 4 * len(e.Enrs)
		for ii := 0; ii < len(e.Enrs); ii++ {
			dst = ssz.WriteOffset(dst, offset)
			offset += len(e.Enrs[ii])
		}
	}
	for ii := 0; ii < len(e.Enrs); ii++ {
		if size := len(e.Enrs[ii]); size > 2048 {
			err = ssz.ErrBytesLengthFn("Enrs.Enrs[ii]", size, 2048)
			return
		}
		dst = append(dst, e.Enrs[ii]...)
	}

	return
}

// UnmarshalSSZ ssz unmarshals the Enrs object
func (e *Enrs) UnmarshalSSZ(buf []byte) error {
	var err error
	tail := buf
	// Field (0) 'Enrs'
	{
		buf = tail[:]
		num, err := ssz.DecodeDynamicLength(buf, 32)
		if err != nil {
			return err
		}
		e.Enrs = make([][]byte, num)
		err = ssz.UnmarshalDynamic(buf, num, func(indx int, buf []byte) (err error) {
			if len(buf) > 2048 {
				return ssz.ErrBytesLength
			}
			if cap(e.Enrs[indx]) == 0 {
				e.Enrs[indx] = make([]byte, 0, len(buf))
			}
			e.Enrs[indx] = append(e.Enrs[indx], buf...)
			return nil
		})
		if err != nil {
			return err
		}
	}
	return err
}

// SizeSSZ returns the ssz encoded size in bytes for the Enrs object
func (e *Enrs) SizeSSZ() (size int) {
	size = 0

	// Field (0) 'Enrs'
	for ii := 0; ii < len(e.Enrs); ii++ {
		size += 4
		size += len(e.Enrs[ii])
	}

	return
}

// HashTreeRoot ssz hashes the Enrs object
func (e *Enrs) HashTreeRoot() ([32]byte, error) {
	return ssz.HashWithDefaultHasher(e)
}

// HashTreeRootWith ssz hashes the Enrs object with a hasher
func (e *Enrs) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	indx := hh.Index()

	// Field (0) 'Enrs'
	{
		subIndx := hh.Index()
		num := uint64(len(e.Enrs))
		if num > 32 {
			err = ssz.ErrIncorrectListSize
			return
		}
		for _, elem := range e.Enrs {
			{
				elemIndx := hh.Index()
				byteLen := uint64(len(elem))
				if byteLen > 2048 {
					err = ssz.ErrIncorrectListSize
					return
				}
				hh.AppendBytes32(elem)
				hh.MerkleizeWithMixin(elemIndx, byteLen, (2048+31)/32)
			}
		}
		hh.MerkleizeWithMixin(subIndx, num, 32)
	}

	hh.Merkleize(indx)
	return
}

// GetTree ssz hashes the Enrs object
func (e *Enrs) GetTree() (*ssz.Node, error) {
	return ssz.ProofTree(e)
}
