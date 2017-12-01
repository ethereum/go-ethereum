package enr

import (
	"fmt"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/rlp"
)

// IP6 represents an 16-byte IPv6 address in a node record.
type IP6 net.IP

// ENRKey returns the node record key for an IPv6 address.
func (IP6) ENRKey() string {
	return "ip6"
}

func (v IP6) EncodeRLP(w io.Writer) error {
	ip6 := net.IP(v)
	return rlp.Encode(w, ip6)
}

func (v *IP6) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 16 {
		return fmt.Errorf("invalid IPv6 address, want 16 bytes: %v", *v)
	}
	return nil
}
