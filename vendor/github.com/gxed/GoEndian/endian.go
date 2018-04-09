package endian

import (
	"encoding/binary"
	"unsafe"
)

//保存机器大小端
var Endian binary.ByteOrder
var bigEndian bool

func IsBigEndian() bool {
	return bigEndian
}

func IsLittleEndian() bool {
	return !bigEndian
}

func init() {
	if getEndian() {
		Endian = binary.BigEndian
		bigEndian = true
	} else {
		Endian = binary.LittleEndian
		bigEndian = false
	}
}

//以下代码判断机器大小端
const INT_SIZE int = int(unsafe.Sizeof(0))

//true = big endian, false = little endian
func getEndian() (ret bool) {
	var i int = 0x1
	bs := (*[INT_SIZE]byte)(unsafe.Pointer(&i))
	if bs[0] == 0 {
		return true
	} else {
		return false
	}

}
