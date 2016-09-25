package boom

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Buckets is a fast, space-efficient array of buckets where each bucket can
// store up to a configured maximum value.
type Buckets struct {
	data       []byte
	bucketSize uint8
	max        uint8
	count      uint
}

// NewBuckets creates a new Buckets with the provided number of buckets where
// each bucket is the specified number of bits.
func NewBuckets(count uint, bucketSize uint8) *Buckets {
	return &Buckets{
		count:      count,
		data:       make([]byte, (count*uint(bucketSize)+7)/8),
		bucketSize: bucketSize,
		max:        (1 << bucketSize) - 1,
	}
}

// MaxBucketValue returns the maximum value that can be stored in a bucket.
func (b *Buckets) MaxBucketValue() uint8 {
	return b.max
}

// Count returns the number of buckets.
func (b *Buckets) Count() uint {
	return b.count
}

// Increment will increment the value in the specified bucket by the provided
// delta. A bucket can be decremented by providing a negative delta. The value
// is clamped to zero and the maximum bucket value. Returns itself to allow for
// chaining.
func (b *Buckets) Increment(bucket uint, delta int32) *Buckets {
	val := int32(b.getBits(bucket*uint(b.bucketSize), uint(b.bucketSize))) + delta
	if val > int32(b.max) {
		val = int32(b.max)
	} else if val < 0 {
		val = 0
	}

	b.setBits(uint32(bucket)*uint32(b.bucketSize), uint32(b.bucketSize), uint32(val))
	return b
}

// Set will set the bucket value. The value is clamped to zero and the maximum
// bucket value. Returns itself to allow for chaining.
func (b *Buckets) Set(bucket uint, value uint8) *Buckets {
	if value > b.max {
		value = b.max
	}

	b.setBits(uint32(bucket)*uint32(b.bucketSize), uint32(b.bucketSize), uint32(value))
	return b
}

// Get returns the value in the specified bucket.
func (b *Buckets) Get(bucket uint) uint32 {
	return b.getBits(bucket*uint(b.bucketSize), uint(b.bucketSize))
}

// Reset restores the Buckets to the original state. Returns itself to allow
// for chaining.
func (b *Buckets) Reset() *Buckets {
	b.data = make([]byte, (b.count*uint(b.bucketSize)+7)/8)
	return b
}

// getBits returns the bits at the specified offset and length.
func (b *Buckets) getBits(offset, length uint) uint32 {
	byteIndex := offset / 8
	byteOffset := offset % 8
	if byteOffset+length > 8 {
		rem := 8 - byteOffset
		return b.getBits(offset, rem) | (b.getBits(offset+rem, length-rem) << rem)
	}
	bitMask := uint32((1 << length) - 1)
	return (uint32(b.data[byteIndex]) & (bitMask << byteOffset)) >> byteOffset
}

// setBits sets bits at the specified offset and length.
func (b *Buckets) setBits(offset, length, bits uint32) {
	byteIndex := offset / 8
	byteOffset := offset % 8
	if byteOffset+length > 8 {
		rem := 8 - byteOffset
		b.setBits(offset, rem, bits)
		b.setBits(offset+rem, length-rem, bits>>rem)
		return
	}
	bitMask := uint32((1 << length) - 1)
	b.data[byteIndex] = byte(uint32(b.data[byteIndex]) & ^(bitMask << byteOffset))
	b.data[byteIndex] = byte(uint32(b.data[byteIndex]) | ((bits & bitMask) << byteOffset))
}

// WriteTo writes a binary representation of Buckets to an i/o stream.
// It returns the number of bytes written.
func (b *Buckets) WriteTo(stream io.Writer) (int64, error) {
	err := binary.Write(stream, binary.BigEndian, b.bucketSize)
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, b.max)
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, uint64(b.count))
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, uint64(len(b.data)))
	if err != nil {
		return 0, err
	}
	err = binary.Write(stream, binary.BigEndian, b.data)
	if err != nil {
		return 0, err
	}
	return int64(len(b.data) + 2*binary.Size(uint8(0)) + 2*binary.Size(uint64(0))), err
}

// ReadFrom reads a binary representation of Buckets (such as might
// have been written by WriteTo()) from an i/o stream. It returns the number
// of bytes read.
func (b *Buckets) ReadFrom(stream io.Reader) (int64, error) {
	var bucketSize, max uint8
	var count, len uint64
	err := binary.Read(stream, binary.BigEndian, &bucketSize)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.BigEndian, &max)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.BigEndian, &count)
	if err != nil {
		return 0, err
	}
	err = binary.Read(stream, binary.BigEndian, &len)
	if err != nil {
		return 0, err
	}
	data := make([]byte, len)
	err = binary.Read(stream, binary.BigEndian, &data)
	if err != nil {
		return 0, err
	}
	b.bucketSize = bucketSize
	b.max = max
	b.count = uint(count)
	b.data = data
	return int64(int(len) + 2*binary.Size(uint8(0)) + 2*binary.Size(uint64(0))), nil
}

// GobEncode implements gob.GobEncoder interface.
func (b *Buckets) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	_, err := b.WriteTo(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// GobDecode implements gob.GobDecoder interface.
func (b *Buckets) GobDecode(data []byte) error {
	buf := bytes.NewBuffer(data)
	_, err := b.ReadFrom(buf)

	return err
}
