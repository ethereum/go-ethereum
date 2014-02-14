package ethutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func NumberToBytes(num interface{}, bits int) []byte {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, num)
	if err != nil {
		fmt.Println("NumberToBytes failed:", err)
	}

	return buf.Bytes()[buf.Len()-(bits/8):]
}

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

// Read variable integer in big endian
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

func BinaryLength(num int) int {
	if num == 0 {
		return 0
	}

	return 1 + BinaryLength(num>>8)
}
