package utils

func ReverseBytesInPlace(src []byte) {
	for i := 0; i < len(src)/2; i++ {
		src[i], src[len(src)-i-1] = src[len(src)-i-1], src[i]
	}
}

func ReverseBytes(src []byte) []byte {
	lenth := len(src)
	dst := make([]byte, lenth)
	for i := 0; i < len(src); i++ {
		dst[lenth-1-i] = src[i]
	}
	return dst
}
