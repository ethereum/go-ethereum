package pogreb

import (
	"encoding/binary"

	"github.com/akrylysov/pogreb/fs"
)

type slot struct {
	hash      uint32
	keySize   uint16
	valueSize uint32
	kvOffset  int64
}

func (sl slot) kvSize() uint32 {
	return uint32(sl.keySize) + sl.valueSize
}

type bucket struct {
	slots [slotsPerBucket]slot
	next  int64
}

type bucketHandle struct {
	bucket
	file   fs.MmapFile
	offset int64
}

const (
	bucketSize uint32 = 512
)

func align512(n uint32) uint32 {
	return (n + 511) &^ 511
}

func (b bucket) MarshalBinary() ([]byte, error) {
	buf := make([]byte, bucketSize)
	data := buf
	for i := 0; i < slotsPerBucket; i++ {
		sl := b.slots[i]
		binary.LittleEndian.PutUint32(buf[:4], sl.hash)
		binary.LittleEndian.PutUint16(buf[4:6], sl.keySize)
		binary.LittleEndian.PutUint32(buf[6:10], sl.valueSize)
		binary.LittleEndian.PutUint64(buf[10:18], uint64(sl.kvOffset))
		buf = buf[18:]
	}
	binary.LittleEndian.PutUint64(buf[:8], uint64(b.next))
	return data, nil
}

func (b *bucket) UnmarshalBinary(data []byte) error {
	for i := 0; i < slotsPerBucket; i++ {
		_ = data[18] // bounds check hint to compiler; see golang.org/issue/14808
		b.slots[i].hash = binary.LittleEndian.Uint32(data[:4])
		b.slots[i].keySize = binary.LittleEndian.Uint16(data[4:6])
		b.slots[i].valueSize = binary.LittleEndian.Uint32(data[6:10])
		b.slots[i].kvOffset = int64(binary.LittleEndian.Uint64(data[10:18]))
		data = data[18:]
	}
	b.next = int64(binary.LittleEndian.Uint64(data[:8]))
	return nil
}

func (b *bucket) del(slotIdx int) {
	i := slotIdx
	for ; i < slotsPerBucket-1; i++ {
		b.slots[i] = b.slots[i+1]
	}
	b.slots[i] = slot{}
}

func (b *bucketHandle) read() error {
	buf, err := b.file.Slice(b.offset, b.offset+int64(bucketSize))
	if err != nil {
		return err
	}
	return b.UnmarshalBinary(buf)
}

func (b *bucketHandle) write() error {
	buf, err := b.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = b.file.WriteAt(buf, b.offset)
	return err
}

type slotWriter struct {
	bucket      *bucketHandle
	slotIdx     int
	prevBuckets []*bucketHandle
}

func (sw *slotWriter) insert(sl slot, db *DB) error {
	if sw.slotIdx == slotsPerBucket {
		nextBucket, err := db.createOverflowBucket()
		if err != nil {
			return err
		}
		sw.bucket.next = nextBucket.offset
		sw.prevBuckets = append(sw.prevBuckets, sw.bucket)
		sw.bucket = nextBucket
		sw.slotIdx = 0
	}
	sw.bucket.slots[sw.slotIdx] = sl
	sw.slotIdx++
	return nil
}

func (sw *slotWriter) write() error {
	for i := len(sw.prevBuckets) - 1; i >= 0; i-- {
		if err := sw.prevBuckets[i].write(); err != nil {
			return err
		}
	}
	return sw.bucket.write()
}
