package enr

import (
	"fmt"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/rlp"
)

// IP4 represents an 4-byte IPv4 address in a node record.
type IP4 net.IP

// ENRKey returns the node record key for an IPv4 address.
func (IP4) ENRKey() string {
	return "ip4"
}

func (v IP4) EncodeRLP(w io.Writer) error {
	ip4 := net.IP(v).To4()
	if ip4 == nil {
		return fmt.Errorf("invalid IPv4 address: %v", v)
	}
	return rlp.Encode(w, ip4)
}

func (v *IP4) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode((*net.IP)(v)); err != nil {
		return err
	}
	if len(*v) != 4 {
		return fmt.Errorf("invalid IPv4 address, want 4 bytes: %v", *v)
	}
	return nil
}
