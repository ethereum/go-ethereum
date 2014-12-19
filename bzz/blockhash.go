/*
The blockhash package implements a hash tree based fixed block size distributed
data storage
The block hash of a byte array is defined as follows:

- if size is no more than BlockSize, it is stored in a single block
  blockhash = sha256(int64(size) + data)

- if size is more than BlockSize*BlockHashCount^l, but no more than BlockSize*
  BlockHashCount^(l+1), the data vector is split into slices of BlockSize*
  BlockHashCount^l length (except the last one).
  blockhash = sha256(int64(size) + blockhash(slice0) + blockhash(slice1) + ...)
*/

package blockhash

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const HashSize = 32
const BlockSize = 4096
const BlockHashCount = BlockSize / HashSize

type HashType []byte

/*
The layered (memory, disk, distributed) storage model provides two channels, one
for storing and one for retrieving blocks. The layers are chained so that every
layer can store blocks and try to retrieve them if the previous layer did not
succeed.
*/

type dpaStorage struct {
	store_chn    chan *dpaStoreReq
	retrieve_chn chan *dpaRetrieveReq
	chain        *dpaStorage
}

type dpaReaderAt struct {
	hash  HashType
	store *dpaStorage
	size  int64
}

type dpaNode struct {
	data []byte
	size int64 // denotes the size of data represented by the whole subtree
}

type dpaStoreReq struct {
	dpaNode
	hash HashType
}

type dpaRetrieveRes struct {
	dpaNode
	req_id int
}

type dpaRetrieveReq struct {
	hash       HashType
	req_id     int
	result_chn chan *dpaRetrieveRes
}

func (h HashType) bits(i, j uint) uint {

	ii := i >> 3
	jj := i & 7
	if ii >= HashSize {
		return 0
	}

	if jj+j <= 8 {
		return uint((h[ii] >> jj) & ((1 << j) - 1))
	}

	res := uint(h[ii] >> jj)
	jj = 8 - jj
	j -= jj
	for j != 0 {
		ii++
		if j < 8 {
			res += uint(h[ii]&((1<<j)-1)) << jj
			return res
		}
		res += uint(h[ii]) << jj
		jj += 8
		j -= 8
	}
	return res
}

func (h HashType) isEqual(h2 HashType) bool {

	for i := range h {
		if h[i] != h2[i] {
			return false
		}
	}
	return true

}

func (s *dpaStorage) Init() {

	s.store_chn = make(chan *dpaStoreReq, 1000)
	s.retrieve_chn = make(chan *dpaRetrieveReq, 1000)

}

// get the root hash of any data vector and store the blocks of the tree if store != nil

func GetDPAroot(data []byte, store *dpaStorage) HashType {

	return GetDPAhash(io.NewSectionReader(bytes.NewReader(data), 0, int64(len(data))), store)

}

func goGetDPAhash(reader *io.SectionReader, store *dpaStorage, hash HashType, done chan<- bool) {

	hh := GetDPAhash(reader, store)
	if hh == nil {
		done <- false
	} else {
		copy(hash[:], hh[:])
		done <- true
	}

}

func GetDPAhash(reader *io.SectionReader, store *dpaStorage) HashType {

	size := reader.Size()
	var block []byte

	if size <= BlockSize {
		block = make([]byte, size)
		br, _ := reader.Read(block)
		if br < int(size) {
			return nil
		}
	} else {
		stc := (size + BlockSize - 1) / BlockSize
		SubtreeSize := int64(BlockSize)

		for stc > BlockHashCount {
			stc = (stc-1)/BlockHashCount + 1
			SubtreeSize *= BlockHashCount
		}
		SubtreeCount := int(stc)

		block = make([]byte, SubtreeCount*HashSize)

		hdone := make(chan bool, SubtreeCount)

		ptr := int64(0)
		hptr := 0
		for i := 0; i < SubtreeCount; i++ {
			ptr2 := ptr + SubtreeSize
			if ptr2 > size {
				ptr2 = size
			}
			go goGetDPAhash(io.NewSectionReader(reader, ptr, ptr2-ptr), store, HashType(block[hptr:hptr+HashSize]), hdone)
			ptr = ptr2
			hptr += HashSize
		}

		for i := 0; i < SubtreeCount; i++ {
			if !<-hdone {
				return nil
			}
		}
	}

	hashfn := sha256.New()
	//binary.LittleEndian.PutUint16(b, uint16(i))
	//fmt.Printf("%d\n", size)
	binary.Write(hashfn, binary.LittleEndian, int64(size))
	hashfn.Write(block)
	hash := hashfn.Sum(nil)

	if store != nil {
		req := new(dpaStoreReq)
		req.data = block
		req.size = int64(size)
		req.hash = hash
		store.store_chn <- req
	}

	return hash

}

// recursive function to retrieve a section of a subtree
// len(data) == stop-start

func getDPAblock(res *dpaRetrieveRes, data []byte, start int64, stop int64, bsize int64, retrv chan<- *dpaRetrieveReq, done chan<- bool) bool {

	for bsize >= res.size {
		if bsize == BlockSize {
			bsize = 0
		} else {
			bsize /= BlockHashCount
		}
	}

	if bsize < BlockSize {
		if res.size < stop {
			if done != nil {
				done <- false
			}
			return false
		}
		copy(data[:], res.data[start:stop])
		if done != nil {
			done <- true
		}
		return true
	}

	bstart := int(start / bsize)
	bstop := int((stop + bsize - 1) / bsize)

	if len(res.data) < bstop*HashSize {
		if done != nil {
			done <- false
		}
		return false
	}

	chn := make(chan *dpaRetrieveRes, bstop-bstart)
	sdone := make(chan bool, bstop-bstart)

	for i := bstart; i < bstop; i++ {

		hash := HashType(res.data[i*HashSize : (i+1)*HashSize])
		req := new(dpaRetrieveReq)
		req.hash = hash
		req.req_id = i
		req.result_chn = chn
		retrv <- req

	}

	for j := bstart; j < bstop; j++ {

		res := <-chn

		i := int64(res.req_id)
		a := i * bsize
		aa := a
		b := a + bsize

		if a < start {
			a = start
		}
		if b > stop {
			b = stop
		}

		if res.size < b-aa {
			if done != nil {
				done <- false
			}
			return false
		}

		if bsize == BlockSize {
			getDPAblock(res, data[a-start:b-start], a-aa, b-aa, 0, retrv, sdone)
		} else {
			go getDPAblock(res, data[a-start:b-start], a-aa, b-aa, bsize/BlockHashCount, retrv, sdone)
		}
	}

	dd := true
	for j := bstart; j < bstop; j++ {
		if !<-sdone {
			dd = false
			break
		}
	}

	if done != nil {
		done <- dd
	}
	return dd

}

func (r *dpaReaderAt) ReadAt(p []byte, off int64) (n int, err error) {

	chn := make(chan *dpaRetrieveRes)

	req := new(dpaRetrieveReq)
	req.hash = r.hash
	req.req_id = 0
	req.result_chn = chn

	r.store.retrieve_chn <- req
	res := <-chn

	if res.size == 0 {
		return 0, fmt.Errorf("Block hash %064x not found", r.hash)
	}

	r.size = res.size
	if len(p) == 0 {
		return 0, nil
	}

	bsize := int64(0)
	if res.size > BlockSize {
		bsize = int64(BlockSize)
		for bsize*BlockHashCount < res.size {
			bsize *= BlockHashCount
		}
	}

	err = error(nil)

	eoff := off + int64(len(p))
	if eoff > res.size {
		eoff = res.size
		err = io.EOF
	}

	if !getDPAblock(res, p, off, eoff, bsize, r.store.retrieve_chn, nil) {
		return 0, fmt.Errorf("Can't load section [%d:%d] of block hash %064x", off, eoff, r.hash)
	}

	return int(eoff - off), err

}

func GetDPAreader(hash HashType, st *dpaStorage) *io.SectionReader {

	rd := new(dpaReaderAt)
	rd.hash = hash
	rd.store = st
	rd.size = -1

	rd.ReadAt(nil, 0)

	if rd.size >= 0 {
		return io.NewSectionReader(rd, 0, rd.size)
	} else {
		return nil
	}

}

// retrieve a data vector of a given block hash from the given storage

func GetDPAdata(hash HashType, st *dpaStorage) []byte {

	sr := GetDPAreader(hash, st)
	if sr == nil {
		return nil
	}

	size := sr.Size()

	data := make([]byte, int(size))
	br, _ := sr.Read(data)
	if int64(br) == size {
		return data
	} else {
		return nil
	}

}
