package chunk

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

const (
	DefaultSize   = 4096
	MaxPO         = 16
	AddressLength = 32
)

var (
	ErrChunkNotFound = errors.New("chunk not found")
	ErrChunkInvalid  = errors.New("invalid chunk")
)

type Chunk interface {
	Address() Address
	Data() []byte
}

type chunk struct {
	addr  Address
	sdata []byte
}

func NewChunk(addr Address, data []byte) Chunk {
	return &chunk{
		addr:  addr,
		sdata: data,
	}
}

func (c *chunk) Address() Address {
	return c.addr
}

func (c *chunk) Data() []byte {
	return c.sdata
}

func (self *chunk) String() string {
	return fmt.Sprintf("Address: %v Chunksize: %v", self.addr.Log(), len(self.sdata))
}

type Address []byte

var ZeroAddr = Address(common.Hash{}.Bytes())

func (a Address) Hex() string {
	return fmt.Sprintf("%064x", []byte(a[:]))
}

func (a Address) Log() string {
	if len(a[:]) < 8 {
		return fmt.Sprintf("%x", []byte(a[:]))
	}
	return fmt.Sprintf("%016x", []byte(a[:8]))
}

func (a Address) String() string {
	return fmt.Sprintf("%064x", []byte(a))
}

func (a Address) MarshalJSON() (out []byte, err error) {
	return []byte(`"` + a.String() + `"`), nil
}

func (a *Address) UnmarshalJSON(value []byte) error {
	s := string(value)
	*a = make([]byte, 32)
	h := common.Hex2Bytes(s[1 : len(s)-1])
	copy(*a, h)
	return nil
}

// Proximity returns the proximity order of the MSB distance between x and y
//
// The distance metric MSB(x, y) of two equal length byte sequences x an y is the
// value of the binary integer cast of the x^y, ie., x and y bitwise xor-ed.
// the binary cast is big endian: most significant bit first (=MSB).
//
// Proximity(x, y) is a discrete logarithmic scaling of the MSB distance.
// It is defined as the reverse rank of the integer part of the base 2
// logarithm of the distance.
// It is calculated by counting the number of common leading zeros in the (MSB)
// binary representation of the x^y.
//
// (0 farthest, 255 closest, 256 self)
func Proximity(one, other []byte) (ret int) {
	b := (MaxPO-1)/8 + 1
	if b > len(one) {
		b = len(one)
	}
	m := 8
	for i := 0; i < b; i++ {
		oxo := one[i] ^ other[i]
		for j := 0; j < m; j++ {
			if (oxo>>uint8(7-j))&0x01 != 0 {
				return i*8 + j
			}
		}
	}
	return MaxPO
}

// ModeGet enumerates different Getter modes.
type ModeGet int

// Getter modes.
const (
	// ModeGetRequest: when accessed for retrieval
	ModeGetRequest ModeGet = iota
	// ModeGetSync: when accessed for syncing or proof of custody request
	ModeGetSync
	// ModeGetLookup: when accessed to lookup a a chunk in feeds or other places
	ModeGetLookup
)

// ModePut enumerates different Putter modes.
type ModePut int

// Putter modes.
const (
	// ModePutRequest: when a chunk is received as a result of retrieve request and delivery
	ModePutRequest ModePut = iota
	// ModePutSync: when a chunk is received via syncing
	ModePutSync
	// ModePutUpload: when a chunk is created by local upload
	ModePutUpload
)

// ModeSet enumerates different Setter modes.
type ModeSet int

// Setter modes.
const (
	// ModeSetAccess: when an update request is received for a chunk or chunk is retrieved for delivery
	ModeSetAccess ModeSet = iota
	// ModeSetSync: when push sync receipt is received
	ModeSetSync
	// ModeSetRemove: when a chunk is removed
	ModeSetRemove
)

// Descriptor holds information required for Pull syncing. This struct
// is provided by subscribing to pull index.
type Descriptor struct {
	Address Address
	BinID   uint64
}

func (d *Descriptor) String() string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("%s bin id %v", d.Address.Hex(), d.BinID)
}

type Store interface {
	Get(ctx context.Context, mode ModeGet, addr Address) (ch Chunk, err error)
	Put(ctx context.Context, mode ModePut, ch Chunk) (exists bool, err error)
	Has(ctx context.Context, addr Address) (yes bool, err error)
	Set(ctx context.Context, mode ModeSet, addr Address) (err error)
	LastPullSubscriptionBinID(bin uint8) (id uint64, err error)
	SubscribePull(ctx context.Context, bin uint8, since, until uint64) (c <-chan Descriptor, stop func())
	Close() (err error)
}

// Validator validates a chunk.
type Validator interface {
	Validate(ch Chunk) bool
}

// ValidatorStore encapsulates Store by decorting the Put method
// with validators check.
type ValidatorStore struct {
	Store
	validators []Validator
}

// NewValidatorStore returns a new ValidatorStore which uses
// provided validators to validate chunks on Put.
func NewValidatorStore(store Store, validators ...Validator) (s *ValidatorStore) {
	return &ValidatorStore{
		Store:      store,
		validators: validators,
	}
}

// Put overrides Store put method with validators check. If one of the validators
// return true, the chunk is considered valid and Store Put method is called.
// If all validators return false, ErrChunkInvalid is returned.
func (s *ValidatorStore) Put(ctx context.Context, mode ModePut, ch Chunk) (exists bool, err error) {
	for _, v := range s.validators {
		if v.Validate(ch) {
			return s.Store.Put(ctx, mode, ch)
		}
	}
	return false, ErrChunkInvalid
}
