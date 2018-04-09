package madns

import (
	"errors"
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
)

var Dns4Protocol = ma.Protocol{
	Code:       54,
	Size:       ma.LengthPrefixedVarSize,
	Name:       "dns4",
	VCode:      ma.CodeToVarint(54),
	Transcoder: DnsTranscoder,
}
var Dns6Protocol = ma.Protocol{
	Code:       55,
	Size:       ma.LengthPrefixedVarSize,
	Name:       "dns6",
	VCode:      ma.CodeToVarint(55),
	Transcoder: DnsTranscoder,
}
var DnsaddrProtocol = ma.Protocol{
	Code:       56,
	Size:       ma.LengthPrefixedVarSize,
	Name:       "dnsaddr",
	VCode:      ma.CodeToVarint(56),
	Transcoder: DnsTranscoder,
}

func init() {
	err := ma.AddProtocol(Dns4Protocol)
	if err != nil {
		panic(fmt.Errorf("error registering dns4 protocol: %s", err))
	}
	err = ma.AddProtocol(Dns6Protocol)
	if err != nil {
		panic(fmt.Errorf("error registering dns6 protocol: %s", err))
	}
	err = ma.AddProtocol(DnsaddrProtocol)
	if err != nil {
		panic(fmt.Errorf("error registering dnsaddr protocol: %s", err))
	}
}

var DnsTranscoder = ma.NewTranscoderFromFunctions(dnsStB, dnsBtS)

func dnsStB(s string) ([]byte, error) {
	size := ma.CodeToVarint(len(s))
	b := append(size, []byte(s)...)
	return b, nil
}

func dnsBtS(b []byte) (string, error) {
	size, n, err := ma.ReadVarintCode(b)
	if err != nil {
		return "", err
	}

	b = b[n:]
	if len(b) != size {
		return "", errors.New("inconsistent lengths")
	}
	return string(b), nil
}
