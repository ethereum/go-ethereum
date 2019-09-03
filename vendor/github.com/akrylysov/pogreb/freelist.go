package pogreb

import (
	"encoding/binary"
	"sort"
)

type block struct {
	offset int64
	size   uint32
}

type freelist struct {
	blocks []block
}

func (fl *freelist) search(size uint32) int {
	return sort.Search(len(fl.blocks), func(i int) bool {
		return fl.blocks[i].size >= size
	})
}

func (fl *freelist) free(off int64, size uint32) {
	if size == 0 {
		panic("unable to free zero bytes")
	}
	i := fl.search(size)
	if i < len(fl.blocks) && off == fl.blocks[i].offset {
		panic("freeing already freed offset")
	}

	fl.blocks = append(fl.blocks, block{})
	copy(fl.blocks[i+1:], fl.blocks[i:])
	fl.blocks[i] = block{offset: off, size: size}
}

func (fl *freelist) allocate(size uint32) int64 {
	if size == 0 {
		panic("unable to allocate zero bytes")
	}
	i := fl.search(size)
	if i >= len(fl.blocks) {
		return -1
	}
	off := fl.blocks[i].offset
	if fl.blocks[i].size == size {
		copy(fl.blocks[i:], fl.blocks[i+1:])
		fl.blocks[len(fl.blocks)-1] = block{}
		fl.blocks = fl.blocks[:len(fl.blocks)-1]
	} else {
		fl.blocks[i].size -= size
		fl.blocks[i].offset += int64(size)
	}
	return off
}

func (fl *freelist) defrag() {
	if len(fl.blocks) <= 1 {
		return
	}
	sort.Slice(fl.blocks, func(i, j int) bool {
		return fl.blocks[i].offset < fl.blocks[j].offset
	})
	var merged []block
	curOff := fl.blocks[0].offset
	curSize := fl.blocks[0].size
	for i := 1; i < len(fl.blocks); i++ {
		if curOff+int64(curSize) == fl.blocks[i].offset {
			curSize += fl.blocks[i].size
		} else {
			merged = append(merged, block{size: curSize, offset: curOff})
			curOff = fl.blocks[i].offset
			curSize = fl.blocks[i].size
		}
	}
	merged = append(merged, block{offset: curOff, size: curSize})
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].size < merged[j].size
	})
	fl.blocks = merged
}

func (fl *freelist) MarshalBinary() ([]byte, error) {
	size := fl.binarySize()
	buf := make([]byte, size)
	data := buf
	binary.LittleEndian.PutUint32(data[:4], uint32(len(fl.blocks)))
	data = data[4:]
	for i := 0; i < len(fl.blocks); i++ {
		binary.LittleEndian.PutUint64(data[:8], uint64(fl.blocks[i].offset))
		binary.LittleEndian.PutUint32(data[8:12], fl.blocks[i].size)
		data = data[12:]
	}
	return buf, nil
}

func (fl *freelist) binarySize() uint32 {
	return uint32(4 + (8+4)*len(fl.blocks)) // FIXME: this is ugly
}

func (fl *freelist) read(f file, off int64) error {
	if off == -1 {
		return nil
	}
	buf := make([]byte, 4)
	if _, err := f.ReadAt(buf, off); err != nil {
		return err
	}
	n := binary.LittleEndian.Uint32(buf)
	buf = make([]byte, (4+8)*n)
	if _, err := f.ReadAt(buf, off+4); err != nil {
		return err
	}
	for i := uint32(0); i < n; i++ {
		blockOff := int64(binary.LittleEndian.Uint64(buf[:8]))
		blockSize := binary.LittleEndian.Uint32(buf[8:12])
		if blockOff != 0 {
			fl.blocks = append(fl.blocks, block{size: blockSize, offset: blockOff})
		}
		buf = buf[12:]
	}
	fl.free(off, align512(4+(4+8)*n))
	return nil
}

func (fl *freelist) write(f file) (int64, error) {
	if len(fl.blocks) == 0 {
		return -1, nil
	}
	marshaledSize := align512(fl.binarySize())
	i := fl.search(marshaledSize)
	var off int64
	if i < len(fl.blocks) {
		off = fl.blocks[i].offset
		fl.blocks[i] = block{}
	} else {
		var err error
		off, err = f.extend(marshaledSize)
		if err != nil {
			return -1, err
		}
	}
	buf, err := fl.MarshalBinary()
	if err != nil {
		return -1, err
	}
	_, err = f.WriteAt(buf, off)
	return off, err
}
