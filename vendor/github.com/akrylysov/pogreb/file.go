package pogreb

import (
	"encoding"
	"os"

	"github.com/akrylysov/pogreb/fs"
)

type file struct {
	fs.MmapFile
	size int64
}

func openFile(fsyst fs.FileSystem, name string, flag int, perm os.FileMode) (file, error) {
	fi, err := fsyst.OpenFile(name, flag, perm)
	f := file{}
	if err != nil {
		return f, err
	}
	f.MmapFile = fi
	stat, err := fi.Stat()
	if err != nil {
		return f, err
	}
	f.size = stat.Size()
	return f, err
}

func (f *file) extend(size uint32) (int64, error) {
	off := f.size
	if err := f.Truncate(off + int64(size)); err != nil {
		return 0, err
	}
	f.size += int64(size)
	return off, f.Mmap(f.size)
}

func (f *file) append(data []byte) (int64, error) {
	off := f.size
	if _, err := f.WriteAt(data, off); err != nil {
		return 0, err
	}
	f.size += int64(len(data))
	return off, f.Mmap(f.size)
}

func (f *file) writeMarshalableAt(m encoding.BinaryMarshaler, off int64) error {
	buf, err := m.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = f.WriteAt(buf, off)
	return err
}

func (f *file) readUnmarshalableAt(m encoding.BinaryUnmarshaler, size uint32, off int64) error {
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, off); err != nil {
		return err
	}
	return m.UnmarshalBinary(buf)
}
