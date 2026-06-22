// Ported verbatim from github.com/QuarkChain/goquarkchain/common (byte-compatible).

package common

import (
	"errors"
	"fmt"
)

var (
	TOKENBASE  = uint64(36)
	TOKENIDMAX = uint64(4873763662273663091) // ZZZZZZZZZZZZ
	TOKENMAX   = "ZZZZZZZZZZZZ"
)

func TokenIDEncode(str string) uint64 {
	if len(str) >= 13 {
		panic(errors.New("name too long"))
	}
	// TODO check name can only contain 0-9, A-Z

	id := TokenCharEncode(str[len(str)-1])
	base := TOKENBASE

	len := len(str)
	for index := len - 2; index >= 0; index-- {
		id += base * (TokenCharEncode(str[index]) + 1)
		base *= TOKENBASE
	}
	return id
}

func TokenIdDecode(id uint64) (string, error) {
	if id > TOKENIDMAX {
		return "", errors.New("it too big or negative")
	}
	name := make([]byte, 0)
	t, err := TokenCharDecode(id % TOKENBASE)
	if err != nil {
		return "", err
	}
	name = append(name, t)
	if id/TOKENBASE < 1 {
		return string(name), nil
	}
	id = id/TOKENBASE - 1
	for id >= 0 {
		t, err := TokenCharDecode(id % TOKENBASE)
		if err != nil {
			return "", err
		}
		name = append(name, t)
		if id/TOKENBASE < 1 {
			break
		}
		id = id/TOKENBASE - 1
	}
	return ReverseString(string(name)), nil
}
func TokenCharEncode(char byte) uint64 {
	if char >= byte('A') && char <= byte('Z') {
		return 10 + uint64(char-byte('A'))
	}
	if char >= byte('0') && char <= byte('9') {
		return uint64(char - byte('0'))
	}
	panic(fmt.Errorf("unknown character %v", byte(char)))
}

func TokenCharDecode(id uint64) (byte, error) {
	if !(id < TOKENBASE && id >= 0) {
		return byte(0), fmt.Errorf("incalid char %v", id)
	}
	if id < 10 {
		return byte('0' + id), nil
	}
	return byte('A' + id - 10), nil
}

func ReverseString(s string) (result string) {
	for _, v := range s {
		result = string(v) + result
	}
	return
}
