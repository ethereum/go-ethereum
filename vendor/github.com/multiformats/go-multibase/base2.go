package multibase

import (
	"fmt"
	"strconv"
	"strings"
)

// binaryEncodeToString takes an array of bytes and returns
// multibase binary representation
func binaryEncodeToString(src []byte) string {
	dst := make([]byte, len(src)*8)
	encodeBinary(dst, src)
	return string(dst)
}

// encodeBinary takes the src and dst bytes and converts each
// byte to their binary rep using power reduction method
func encodeBinary(dst []byte, src []byte) {
	for i, b := range src {
		for j := 0; j < 8; j++ {
			if b&(1<<uint(7-j)) == 0 {
				dst[i*8+j] = '0'
			} else {
				dst[i*8+j] = '1'
			}
		}
	}
}

// decodeBinaryString takes multibase binary representation
// and returns a byte array
func decodeBinaryString(s string) ([]byte, error) {
	if len(s)&7 != 0 {
		// prepend the padding
		s = strings.Repeat("0", 8-len(s)&7) + s
	}

	data := make([]byte, len(s)>>3)

	for i, dstIndex := 0, 0; i < len(s); i = i + 8 {
		value, err := strconv.ParseInt(s[i:i+8], 2, 0)
		if err != nil {
			return nil, fmt.Errorf("error while conversion: %s", err)
		}

		data[dstIndex] = byte(value)
		dstIndex++
	}

	return data, nil
}
