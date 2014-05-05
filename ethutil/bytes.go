package ethutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Number to bytes
//
// Returns the number in bytes with the specified base
func NumberToBytes(num interface{}, bits int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		fmt.Println("NumberToBytes failed:", err)
	}

	return buf.Bytes()[buf.Len()-(bits/8):]
}

// Bytes to number
//
// Attempts to cast a byte slice to a unsigned integer
func BytesToNumber(b []byte) uint64 {
	var number uint64

	// Make sure the buffer is 64bits
	data := make([]byte, 8)
	data = append(data[:len(b)], b...)

	buf := bytes.NewReader(data)
	err := binary.Read(buf, binary.BigEndian, &number)
	if err != nil {
		fmt.Println("BytesToNumber failed:", err)
	}

	return number
}

// Read variable int
//
// Read a variable length number in big endian byte order
func ReadVarint(reader *bytes.Reader) (ret uint64) {
	if reader.Len() == 8 {
		var num uint64
		binary.Read(reader, binary.BigEndian, &num)
		ret = uint64(num)
	} else if reader.Len() == 4 {
		var num uint32
		binary.Read(reader, binary.BigEndian, &num)
		ret = uint64(num)
	} else if reader.Len() == 2 {
		var num uint16
		binary.Read(reader, binary.BigEndian, &num)
		ret = uint64(num)
	} else {
		var num uint8
		binary.Read(reader, binary.BigEndian, &num)
		ret = uint64(num)
	}

	return ret
}

// Binary length
//
// Returns the true binary length of the given number
func BinaryLength(num int) int {
	if num == 0 {
		return 0
	}

	return 1 + BinaryLength(num>>8)
}

// Copy bytes
//
// Returns an exact copy of the provided bytes
func CopyBytes(b []byte) (copiedBytes []byte) {
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

func IsHex(str string) bool {
	l := len(str)
	return l >= 4 && l%2 == 0 && str[0:2] == "0x"
}
