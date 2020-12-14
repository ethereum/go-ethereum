package fixed

// LegacyKeccak20 calculates the LegacyKeccak256 hash of data
func LegacyKeccak20(data [20]byte) [32]byte {
	var a [25]uint64
	// Pad and permute in one go
	a[16] = 0x8000000000000000
	a[2] = 1<<32 | uint64(data[19])<<24 | uint64(data[18])<<16 | uint64(data[17])<<8 | uint64(data[16])
	a[1] = uint64(data[15])<<56 | uint64(data[14])<<48 | uint64(data[13])<<40 | uint64(data[12])<<32 |
		uint64(data[11])<<24 | uint64(data[10])<<16 | uint64(data[9])<<8 | uint64(data[8])
	a[0] = uint64(data[7])<<56 | uint64(data[6])<<48 | uint64(data[5])<<40 | uint64(data[4])<<32 |
		uint64(data[3])<<24 | uint64(data[2])<<16 | uint64(data[1])<<8 | uint64(data[0])
	// Hash it
	keccakF1600(&a)
	// Convert back to bytes, and return
	var buf [32]byte
	/*
		It is possible (the golang lib version does it) to use an unaligned copy here,
		but it is platform specific.
		Since it's a pretty small slice we're dealing with, we're a generic method instead.
		The 'cost' of using the generic version is ~5%, 454ns instead of 430ns.

		ab := (*[136]uint8)(unsafe.Pointer(&a[0]))
		copy(buf[:], ab[:])
	*/
	for i := 0; i < 4; i++ {
		buf[8*i+7] = byte(a[i] >> 56)
		buf[8*i+6] = byte(a[i] >> 48)
		buf[8*i+5] = byte(a[i] >> 40)
		buf[8*i+4] = byte(a[i] >> 32)
		buf[8*i+3] = byte(a[i] >> 24)
		buf[8*i+2] = byte(a[i] >> 16)
		buf[8*i+1] = byte(a[i] >> 8)
		buf[8*i] = byte(a[i])
	}
	return buf
}

// LegacyKeccak32 calculates the LegacyKeccak256 hash of data
func LegacyKeccak32(data [32]byte) [32]byte {
	var a [25]uint64
	// Pad and permute in one go
	a[16] = 0x8000000000000000
	a[4] = 1
	a[3] = uint64(data[31])<<56 | uint64(data[30])<<48 | uint64(data[29])<<40 | uint64(data[28])<<32 |
		uint64(data[27])<<24 | uint64(data[26])<<16 | uint64(data[25])<<8 | uint64(data[24])
	a[2] = uint64(data[23])<<56 | uint64(data[22])<<48 | uint64(data[21])<<40 | uint64(data[20])<<32 |
		uint64(data[19])<<24 | uint64(data[18])<<16 | uint64(data[17])<<8 | uint64(data[16])
	a[1] = uint64(data[15])<<56 | uint64(data[14])<<48 | uint64(data[13])<<40 | uint64(data[12])<<32 |
		uint64(data[11])<<24 | uint64(data[10])<<16 | uint64(data[9])<<8 | uint64(data[8])
	a[0] = uint64(data[7])<<56 | uint64(data[6])<<48 | uint64(data[5])<<40 | uint64(data[4])<<32 |
		uint64(data[3])<<24 | uint64(data[2])<<16 | uint64(data[1])<<8 | uint64(data[0])
	// Hash it
	keccakF1600(&a)
	// Convert back to bytes, and return
	var buf [32]byte
	/*
		It is possible (the golang lib version does it) to use an unaligned copy here,
		but it is platform specific.
		Since it's a pretty small slice we're dealing with, we're a generic method instead.
		The 'cost' of using the generic version is ~5%, 454ns instead of 430ns.

		ab := (*[136]uint8)(unsafe.Pointer(&a[0]))
		copy(buf[:], ab[:])
	*/
	for i := 0; i < 4; i++ {
		buf[8*i+7] = byte(a[i] >> 56)
		buf[8*i+6] = byte(a[i] >> 48)
		buf[8*i+5] = byte(a[i] >> 40)
		buf[8*i+4] = byte(a[i] >> 32)
		buf[8*i+3] = byte(a[i] >> 24)
		buf[8*i+2] = byte(a[i] >> 16)
		buf[8*i+1] = byte(a[i] >> 8)
		buf[8*i] = byte(a[i])
	}
	return buf
}
