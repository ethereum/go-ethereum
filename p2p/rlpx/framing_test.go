// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rlpx

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestPacketReader(t *testing.T) {
	feed := func(pr *packetReader, n uint32) {
		for sent := uint32(0); sent < n; {
			chunk := randomChunk(sent, n-sent)
			sent += uint32(len(chunk))
			if err := pr.bufSema.waitAcquire(uint32(len(chunk)), 200*time.Millisecond); err != nil {
				panic(err.Error())
			}
			end, err := pr.feed(chunk)
			if err != nil {
				panic(fmt.Errorf("pr.feed returned error: %v", err))
			}
			if end && sent != n {
				panic(fmt.Errorf("pr.feed returned end=true with %d/%d bytes of input", sent, n))
			}
		}
	}
	for size := uint32(1); size < 2<<17; size *= 2 {
		sem := newBufSema(staticFrameSize * 2)
		pr := newPacketReader(sem, size, nil)
		go feed(pr, size)
		if err := checkSeq(pr, size); err != nil {
			t.Fatalf("size %d: read error: %v", size, err)
		}
		if val := sem.get(); val != staticFrameSize*2 {
			t.Fatalf("size %d: wrong semaphore value after reading all data. got %d, want %d", size, val, staticFrameSize*2)
		}
	}
}

func checkSeq(r io.Reader, size uint32) error {
	content := make([]byte, size)
	if _, err := io.ReadFull(r, content); err != nil {
		return err
	}
	for i, b := range content {
		if b != byte(i) {
			return fmt.Errorf("mismatch at index %d: have %d, want %d", i, b, byte(i))
		}
	}
	return nil
}

func randomChunk(seed uint32, maxSize uint32) frameBuffer {
	size := rand.Uint32()%staticFrameSize + 1
	if size > maxSize {
		size = maxSize
	}
	chunk := make(frameBuffer, size)
	for i := range chunk {
		chunk[i] = byte(uint32(i) + seed)
	}
	return chunk
}

/*
func TestFrameFakeGolden(t *testing.T) {
	buf := new(bytes.Buffer)
	hash := fakeHash{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	rw := newFrameRW(buf, secrets{
		AES:        crypto.Sha3(),
		MAC:        crypto.Sha3(),
		IngressMAC: hash,
		EgressMAC:  hash,
	})

	golden := hexb(`
00828ddae471818bb0bfa6b551d1cb42
01010101010101010101010101010101
ba628a4ba590cb43f7848f41c4382885
01010101010101010101010101010101
`)
	body := hexb(`08C401020304`)

	// Check sendFrame. This encodes the frame to buf.
	fwbuf := makeFrameWriteBuffer()
	fwbuf.Write(body)
	if err := rw.sendFrame(regularHeader{0, 0}, fwbuf); err != nil {
		t.Fatalf("sendFrame error: %v", err)
	}
	written := buf.Bytes()
	if !bytes.Equal(written, golden) {
		t.Fatalf("output mismatch:\n  got:  %x\n  want: %x", written, golden)
	}

	// Check readFrame. It reads the message encoded by sendFrame, which
	// must be equivalent to the golden message above.
	fsize, hdr, err := rw.readFrameHeader()
	if err != nil {
		t.Fatalf("readFrameHeader error: %v", err)
	}
	if (hdr != frameHeader{}) {
		t.Errorf("read header mismatch: got %v, want zero header", hdr)
	}
	if int(fsize) != len(body) {
		t.Errorf("read size mismatch: got %d, want %d", fsize, len(body))
	}
	// if !bytes.Equal(bodybuf.Bytes(), body) {
	// 	t.Errorf("read body mismatch:\ngot  %x\nwant %x", bodybuf.Bytes(), body)
	// }
}

type fakeHash []byte

func (fakeHash) Write(p []byte) (int, error) { return len(p), nil }
func (fakeHash) Reset()                      {}
func (fakeHash) BlockSize() int              { return 0 }

func (h fakeHash) Size() int           { return len(h) }
func (h fakeHash) Sum(b []byte) []byte { return append(b, h...) }

*/

func hexb(str string) []byte {
	unspace := strings.NewReplacer("\n", "", "\t", "", " ", "")
	b, err := hex.DecodeString(unspace.Replace(str))
	if err != nil {
		panic(fmt.Sprintf("invalid hex string: %q", str))
	}
	return b
}

func hexkey(str string) *ecdsa.PrivateKey {
	return crypto.ToECDSA(hexb(str))
}
