package derivationpath

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

func Encode(rawPath []uint32) string {
	segments := []string{string(tokenMaster)}

	for _, i := range rawPath {
		suffix := ""

		if i >= hardenedStart {
			i = i - hardenedStart
			suffix = string(tokenHardened)
		}

		segments = append(segments, fmt.Sprintf("%d%s", i, suffix))
	}

	return strings.Join(segments, string(tokenSeparator))
}

func EncodeFromBytes(data []byte) (string, error) {
	buf := bytes.NewBuffer(data)
	rawPath := make([]uint32, buf.Len()/4)
	err := binary.Read(buf, binary.BigEndian, &rawPath)
	if err != nil {
		return "", err
	}

	return Encode(rawPath), nil
}
