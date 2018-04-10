package multiaddr

import (
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	mh "github.com/multiformats/go-multihash"
)

type Transcoder interface {
	StringToBytes(string) ([]byte, error)
	BytesToString([]byte) (string, error)
}

func NewTranscoderFromFunctions(s2b func(string) ([]byte, error),
	b2s func([]byte) (string, error)) Transcoder {

	return twrp{s2b, b2s}
}

type twrp struct {
	strtobyte func(string) ([]byte, error)
	bytetostr func([]byte) (string, error)
}

func (t twrp) StringToBytes(s string) ([]byte, error) {
	return t.strtobyte(s)
}
func (t twrp) BytesToString(b []byte) (string, error) {
	return t.bytetostr(b)
}

var TranscoderIP4 = NewTranscoderFromFunctions(ip4StB, ipBtS)
var TranscoderIP6 = NewTranscoderFromFunctions(ip6StB, ipBtS)

func ip4StB(s string) ([]byte, error) {
	i := net.ParseIP(s).To4()
	if i == nil {
		return nil, fmt.Errorf("failed to parse ip4 addr: %s", s)
	}
	return i, nil
}

func ip6StB(s string) ([]byte, error) {
	i := net.ParseIP(s).To16()
	if i == nil {
		return nil, fmt.Errorf("failed to parse ip4 addr: %s", s)
	}
	return i, nil
}

func ipBtS(b []byte) (string, error) {
	return net.IP(b).String(), nil
}

var TranscoderPort = NewTranscoderFromFunctions(portStB, portBtS)

func portStB(s string) ([]byte, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse port addr: %s", err)
	}
	if i >= 65536 {
		return nil, fmt.Errorf("failed to parse port addr: %s", "greater than 65536")
	}
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(i))
	return b, nil
}

func portBtS(b []byte) (string, error) {
	i := binary.BigEndian.Uint16(b)
	return strconv.Itoa(int(i)), nil
}

var TranscoderOnion = NewTranscoderFromFunctions(onionStB, onionBtS)

func onionStB(s string) ([]byte, error) {
	addr := strings.Split(s, ":")
	if len(addr) != 2 {
		return nil, fmt.Errorf("failed to parse onion addr: %s does not contain a port number.", s)
	}

	// onion address without the ".onion" substring
	if len(addr[0]) != 16 {
		return nil, fmt.Errorf("failed to parse onion addr: %s not a Tor onion address.", s)
	}
	onionHostBytes, err := base32.StdEncoding.DecodeString(strings.ToUpper(addr[0]))
	if err != nil {
		return nil, fmt.Errorf("failed to decode base32 onion addr: %s %s", s, err)
	}

	// onion port number
	i, err := strconv.Atoi(addr[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse onion addr: %s", err)
	}
	if i >= 65536 {
		return nil, fmt.Errorf("failed to parse onion addr: %s", "port greater than 65536")
	}
	if i < 1 {
		return nil, fmt.Errorf("failed to parse onion addr: %s", "port less than 1")
	}

	onionPortBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(onionPortBytes, uint16(i))
	bytes := []byte{}
	bytes = append(bytes, onionHostBytes...)
	bytes = append(bytes, onionPortBytes...)
	return bytes, nil
}

func onionBtS(b []byte) (string, error) {
	addr := strings.ToLower(base32.StdEncoding.EncodeToString(b[0:10]))
	port := binary.BigEndian.Uint16(b[10:12])
	return addr + ":" + strconv.Itoa(int(port)), nil
}

var TranscoderIPFS = NewTranscoderFromFunctions(ipfsStB, ipfsBtS)

func ipfsStB(s string) ([]byte, error) {
	// the address is a varint prefixed multihash string representation
	m, err := mh.FromB58String(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ipfs addr: %s %s", s, err)
	}
	size := CodeToVarint(len(m))
	b := append(size, m...)
	return b, nil
}

func ipfsBtS(b []byte) (string, error) {
	// the address is a varint-prefixed multihash string representation
	size, n, err := ReadVarintCode(b)
	if err != nil {
		return "", err
	}

	b = b[n:]
	if len(b) != size {
		return "", errors.New("inconsistent lengths")
	}
	m, err := mh.Cast(b)
	if err != nil {
		return "", err
	}
	return m.B58String(), nil
}

var TranscoderUnix = NewTranscoderFromFunctions(unixStB, unixBtS)

func unixStB(s string) ([]byte, error) {
	// the address is the whole remaining string, prefixed by a varint len
	size := CodeToVarint(len(s))
	b := append(size, []byte(s)...)
	return b, nil
}

func unixBtS(b []byte) (string, error) {
	// the address is a varint len prefixed string
	size, n, err := ReadVarintCode(b)
	if err != nil {
		return "", err
	}

	b = b[n:]
	if len(b) != size {
		return "", errors.New("inconsistent lengths")
	}
	if size == 0 {
		return "", errors.New("invalid length")
	}
	s := string(b)
	s = s[1:] // remove starting slash
	return s, nil
}
