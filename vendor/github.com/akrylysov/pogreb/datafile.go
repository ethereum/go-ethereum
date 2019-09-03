package pogreb

type dataFile struct {
	file
	fl freelist
}

func (f *dataFile) readKeyValue(sl slot) ([]byte, []byte, error) {
	keyValue, err := f.Slice(sl.kvOffset, sl.kvOffset+int64(sl.kvSize()))
	if err != nil {
		return nil, nil, err
	}
	return keyValue[:sl.keySize], keyValue[sl.keySize:], nil
}

func (f *dataFile) readKey(sl slot) ([]byte, error) {
	return f.Slice(sl.kvOffset, sl.kvOffset+int64(sl.keySize))
}

func (f *dataFile) allocate(size uint32) (int64, error) {
	size = align512(size)
	if off := f.fl.allocate(size); off > 0 {
		return off, nil
	}
	return f.extend(size)
}

func (f *dataFile) free(size uint32, off int64) {
	size = align512(size)
	f.fl.free(off, size)
}

func (f *dataFile) writeKeyValue(key []byte, value []byte) (int64, error) {
	dataLen := align512(uint32(len(key) + len(value)))
	data := make([]byte, dataLen)
	copy(data, key)
	copy(data[len(key):], value)
	off := f.fl.allocate(dataLen)
	if off != -1 {
		if _, err := f.WriteAt(data, off); err != nil {
			return 0, err
		}
	} else {
		return f.append(data)
	}
	return off, nil
}
