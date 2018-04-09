// +build gofuzz

package msgio

import "bytes"

// get the go-fuzz tools and build a fuzzer
// $ go get -u github.com/dvyukov/go-fuzz/...
// $ go-fuzz-build github.com/libp2p/go-msgio

// put a corpus of random (even better if actual, structured) data in a corpus directry
// $ go-fuzz -bin ./msgio-fuzz -corpus corpus -workdir=wdir -timeout=15

func Fuzz(data []byte) int {
	rc := NewReader(bytes.NewReader(data))
	// rc := NewVarintReader(bytes.NewReader(data))

	if _, err := rc.ReadMsg(); err != nil {
		return 0
	}

	return 1
}
