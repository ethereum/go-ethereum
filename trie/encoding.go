package trie

import (
	"bytes"
	"encoding/hex"
	"strings"
)

func CompactEncode(hexSlice []byte) string {
	terminator := 0
	if hexSlice[len(hexSlice)-1] == 16 {
		terminator = 1
	}

	if terminator == 1 {
		hexSlice = hexSlice[:len(hexSlice)-1]
	}

	oddlen := len(hexSlice) % 2
	flags := byte(2*terminator + oddlen)
	if oddlen != 0 {
		hexSlice = append([]byte{flags}, hexSlice...)
	} else {
		hexSlice = append([]byte{flags, 0}, hexSlice...)
	}

	var buff bytes.Buffer
	for i := 0; i < len(hexSlice); i += 2 {
		buff.WriteByte(byte(16*hexSlice[i] + hexSlice[i+1]))
	}

	return buff.String()
}

func CompactDecode(str string) []byte {
	base := CompactHexDecode(str)
	base = base[:len(base)-1]
	if base[0] >= 2 {
		base = append(base, 16)
	}
	if base[0]%2 == 1 {
		base = base[1:]
	} else {
		base = base[2:]
	}

	return base
}

func CompactHexDecode(str string) []byte {
	base := "0123456789abcdef"
	var hexSlice []byte

	enc := hex.EncodeToString([]byte(str))
	for _, v := range enc {
		hexSlice = append(hexSlice, byte(strings.IndexByte(base, byte(v))))
	}
	hexSlice = append(hexSlice, 16)

	return hexSlice
}

func DecodeCompact(key []byte) string {
	const base = "0123456789abcdef"
	var str string

	for _, v := range key {
		if v < 16 {
			str += string(base[v])
		}
	}

	res, _ := hex.DecodeString(str)

	return string(res)
}
