package mask

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

var ErrInvalidFormat = errors.New("invalid multiaddr-filter format")

func NewMask(a string) (*net.IPNet, error) {
	parts := strings.Split(a, "/")

	if parts[0] != "" {
		return nil, ErrInvalidFormat
	}

	if len(parts) != 5 {
		return nil, ErrInvalidFormat
	}

	// check it's a valid filter address. ip + cidr
	isip := parts[1] == "ip4" || parts[1] == "ip6"
	iscidr := parts[3] == "ipcidr"
	if !isip || !iscidr {
		return nil, ErrInvalidFormat
	}

	_, ipn, err := net.ParseCIDR(parts[2] + "/" + parts[4])
	if err != nil {
		return nil, err
	}
	return ipn, nil
}

func ConvertIPNet(n *net.IPNet) (string, error) {
	b, _ := n.Mask.Size()
	switch {
	case n.IP.To4() != nil:
		return fmt.Sprintf("/ip4/%s/ipcidr/%d", n.IP, b), nil
	case n.IP.To16() != nil:
		return fmt.Sprintf("/ip6/%s/ipcidr/%d", n.IP, b), nil
	default:
		return "", fmt.Errorf("was not given valid ip addr")
	}
}
