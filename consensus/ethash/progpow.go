package ethash

import (
    "encoding/binary"
    "github.com/ethereum/go-ethereum/crypto/sha3"
)

const (
    progpowCacheWords = 4*1024 // Total size 16*1024 bytes
    progpowLanes      = 32
    progpowRegs       = 16
    progpowCntCache   = 8
    progpowCntMath    = 8
    progpowCntMem     = loopAccesses
    progpowMixBytes   = 2*mixBytes
)

func progpowLight(size uint64, cache []uint32, hash []byte, nonce uint64,
                  blockNumber uint64) ([]byte, []byte) {
    keccak512 := makeHasher(sha3.NewKeccak512())

    lookup := func(index uint32) uint32 {
        rawData := generateDatasetItem(cache, index/16, keccak512)
        data := binary.LittleEndian.Uint32(rawData[(index%16)*4:])
        return data
    }
    return progpow(hash, nonce, size, blockNumber, lookup)
}

func progpowFull(dataset []uint32, hash []byte, nonce uint64,
                 blockNumber uint64) ([]byte, []byte) {
    lookup := func(index uint32) uint32 {
        return dataset[index]
    }
    return progpow(hash, nonce, uint64(len(dataset))*4, blockNumber, lookup)
}


func rotl32(x uint32, n uint32) uint32 {
    return (((x) << (n%32)) | ((x) >> (32 - (n%32))))
}

func rotr32(x uint32, n uint32) uint32 {
    return (((x) >> (n%32)) | ((x) << (32 - (n%32))))
}

func lower32(in uint64) uint32 {
    return uint32(in & uint64(0x00000000FFFFFFFF))
}

func higher32(in uint64) uint32 {
    return uint32((in >> 32) & uint64(0x00000000FFFFFFFF))
}

var keccakfRNDC = [24] uint32 {
    0x00000001, 0x00008082, 0x0000808a, 0x80008000, 0x0000808b, 0x80000001,
    0x80008081, 0x00008009, 0x0000008a, 0x00000088, 0x80008009, 0x8000000a,
    0x8000808b, 0x0000008b, 0x00008089, 0x00008003, 0x00008002, 0x00000080,
    0x0000800a, 0x8000000a, 0x80008081, 0x00008080, 0x80000001, 0x80008008 }

func keccakF800Round(st [25]uint32, r int) [25]uint32 {
    var keccakfROTC = [24]uint32 {  1,  3, 6,  10, 15, 21, 28, 36, 45, 55,  2,
                                   14, 27, 41, 56,  8, 25, 43, 62, 18, 39, 61,
                                   20, 44 }
    var  keccakfPILN = [24]uint32 { 10,  7, 11, 17, 18, 3, 5,  16, 8,  21, 24,
                                     4, 15, 23, 19, 13, 12, 2, 20, 14, 22,  9,
                                     6,  1 }
    bc := make([]uint32, 5)
    // Theta
    for i := 0; i < 5; i++ {
        bc[i] = st[i] ^ st[i + 5] ^ st[i + 10] ^ st[i + 15] ^ st[i + 20]
    }

    for i := 0; i < 5; i++ {
        t := bc[(i + 4) % 5] ^ rotl32(bc[(i + 1) % 5], 1)
        for j := 0; j < 25; j += 5 {
            st[j + i] ^= t
        }
    }

    // Rho Pi
    t := st[1];
    for i := 0; i < 24; i++ {
        j := keccakfPILN[i]
        bc[0] = st[j]
        st[j] = rotl32(t, keccakfROTC[i])
        t = bc[0]
    }

    //  Chi
    for j := 0; j < 25; j += 5 {
        for i := 0; i < 5; i++ {
            bc[i] = st[j + i];
        }
        for i := 0; i < 5; i++ {
            st[j + i] ^= (^bc[(i + 1) % 5]) & bc[(i + 2) % 5];
        }
    }

    //  Iota
    st[0] ^= keccakfRNDC[r];
    return st
}

func keccakF800Short(headerHash []byte, nonce uint64, result []uint32) uint64 {
    var st [25]uint32

    for i := 0; i < 25; i++ {
        st[i] = 0
    }

    for i := 0; i < 8; i++ {
        st[i] = (uint32(headerHash[4*i])) +
                (uint32(headerHash[4*i+1]) << 8) +
                (uint32(headerHash[4*i+2]) << 16) +
                (uint32(headerHash[4*i+3]) << 24)
    }

    st[8] = lower32(nonce)
    st[9] = higher32(nonce)
    for i := 0; i < 8; i++ {
        st[10+i] = result[i]
    }
    for r := 0; r < 21; r++ {
        st = keccakF800Round(st, r)
    }
    st = keccakF800Round(st, 21)
    return (uint64(st[0]) << 32) | uint64(st[1])
}

func keccakF800Long(headerHash []byte, nonce uint64, result []uint32) []byte {
    var st [25]uint32

    for i := 0; i < 25; i++ {
        st[i] = 0
    }
    for i := 0; i < 8; i++ {
        st[i] = (uint32(headerHash[4*i])) +
                (uint32(headerHash[4*i+1]) << 8) +
                (uint32(headerHash[4*i+2]) << 16) +
                (uint32(headerHash[4*i+3]) << 24)
    }

    st[8] = lower32(nonce)
    st[9] = higher32(nonce)
    for i := 0; i < 8; i++ {
        st[10+i] = result[i]
    }
    for r := 0; r < 21; r++ {
        st = keccakF800Round(st, r)
    }
    st = keccakF800Round(st, 21)
    ret := make([]byte, 32)
    for i := 0; i < 8; i++ {
        binary.LittleEndian.PutUint32(ret[i*4:], st[i])
    }
    return ret
}

func fnv1a(h *uint32 , d uint32) uint32 {
    *h = (*h ^ d) * uint32(0x1000193)
    return *h
}

type kiss99State struct {
    z     uint32
    w     uint32
    jsr   uint32
    jcong uint32
}

func kiss99(st *kiss99State) uint32 {
    var MWC uint32
    st.z = 36969 * (st.z & 65535) + (st.z >> 16);
    st.w = 18000 * (st.w & 65535) + (st.w >> 16);
    MWC = ((st.z << 16) + st.w);
    st.jsr ^= (st.jsr << 17)
    st.jsr ^= (st.jsr >> 13)
    st.jsr ^= (st.jsr << 5)
    st.jcong = 69069 * st.jcong + 1234567
    return ((MWC^st.jcong) + st.jsr);
}

func fillMix(seed uint64, laneId uint32) [progpowRegs] uint32 {
    var st  kiss99State
    var mix [progpowRegs]uint32

    fnvHash := uint32(0x811c9dc5)

    st.z     = fnv1a(&fnvHash, lower32(seed))
    st.w     = fnv1a(&fnvHash, higher32(seed))
    st.jsr   = fnv1a(&fnvHash, laneId)
    st.jcong = fnv1a(&fnvHash, laneId)

    for i := 0; i < progpowRegs; i++ {
        mix[i] = kiss99(&st)
    }
    return mix
}

func clz(a uint32) uint32 {
    for i := uint32(0); i < 32; i++ {
        if (a >> (31 - i)) > 0 {
            return i
        }
    }
    return uint32(32)
}

func popcount(a uint32) uint32 {
    count := uint32(0)
    for i := uint32(0); i < 32; i++ {
        if ((a >> (31 - i)) & uint32(1)) == uint32(1) {
            count += 1
        }
    }
    return count
}

// Merge new data from b into the value in a
// Assuming A has high entropy only do ops that retain entropy
// even if B is low entropy
// (IE don't do A&B)
func merge(a *uint32, b uint32, r uint32) {
    switch (r % 4) {
        case 0:
            *a = (*a * 33) + b
        case 1:
            *a = (*a ^ b) * 33
        case 2:
            *a = rotl32(*a, ((r >> 16) % 32)) ^ b
        case 3:
            *a = rotr32(*a, ((r >> 16) % 32)) ^ b
    }
}

func progpowInit(seed uint64) (kiss99State, [progpowRegs] uint32) {
    var randState kiss99State
    var mixSeq [progpowRegs] uint32

    fnvHash := uint32(0x811c9dc5)

    randState.z     = fnv1a(&fnvHash, lower32(seed))
    randState.w     = fnv1a(&fnvHash, higher32(seed))
    randState.jsr   = fnv1a(&fnvHash, lower32(seed))
    randState.jcong = fnv1a(&fnvHash, higher32(seed))

    // Create a random sequence of mix destinations for merge()
    // guaranteeing every location is touched once
    // Uses Fisher CYates shuffle
    for i := uint32(0); i < progpowRegs; i++ {
        mixSeq[i] = i
    }
    for i := uint32(progpowRegs - 1); i > 0; i-- {
        j := kiss99(&randState) % (i + 1)
        temp := mixSeq[i]
        mixSeq[i] = mixSeq[j]
        mixSeq[j] = temp
    }
    return randState, mixSeq;
}

// Random math between two input values
func progpowMath(a uint32, b uint32, r uint32) uint32 {
    switch (r % 11) {
    case 0: return a + b
    case 1: return a * b
    case 2: return higher32(uint64(a)*uint64(b))
    case 3:
        if a < b {
            return a
        }
        return b
    case 4: return rotl32(a, b)
    case 5: return rotr32(a, b)
    case 6: return a & b
    case 7: return a | b
    case 8: return a ^ b
    case 9: return clz(a) + clz(b)
    case 10: return popcount(a) + popcount(b)
    default: return 0
    }
    return 0
}

func progpowLoop(seed uint64, loop uint32,
                 mix *[progpowLanes][progpowRegs]uint32,
                 lookup func(index uint32) uint32,
                 cDag []uint32, datasetSize uint32) {
    // All lanes share a base address for the global load
    // Global offset uses mix[0] to guarantee it depends on the load result
    gOffset := mix[loop%progpowLanes][0] % datasetSize
    gOffset = gOffset*progpowLanes
    iMax := uint32(0)
    // Lanes can execute in parallel and will be convergent
    for l := uint32(0); l < progpowLanes; l++ {
        mixSeqCnt := uint32(0)

        // global load to sequential locations
        data64 := uint64(lookup(2*(gOffset +l)+1)) << 32 |
                  uint64(lookup(2*(gOffset + l)))

        // initialize the seed and mix destination sequence
        randState, mixSeq := progpowInit(seed)

        if progpowCntCache > progpowCntMath {
            iMax = progpowCntCache
        } else {
            iMax = progpowCntMath
        }

        for i := uint32(0); i < iMax; i++ {
            if i < progpowCntCache {
                // Cached memory access
                // lanes access random location
                src1 := kiss99(&randState) % progpowRegs
                offset := mix[l][src1] % progpowCacheWords;
                data32 := cDag[offset];
                dest := mixSeq[mixSeqCnt % progpowRegs]
                mixSeqCnt++
                r := kiss99(&randState)
                merge(&mix[l][dest], data32, r)
            }

            if i < progpowCntMath {
                // Random Math
                src11 := kiss99(&randState)
                src1 := src11 % progpowRegs
                src2 := kiss99(&randState) % progpowRegs
                r1 := kiss99(&randState)
                r2 := kiss99(&randState)
                dest := mixSeq[mixSeqCnt % progpowRegs]
                mixSeqCnt++
                data32 := progpowMath(mix[l][src1], mix[l][src2], r1)
                merge(&mix[l][dest], data32, r2);
            }
        }

        r1 := kiss99(&randState)
        r2 := kiss99(&randState)

        merge(&mix[l][0], lower32(data64), r1)
        dest := mixSeq[mixSeqCnt % progpowRegs]
        mixSeqCnt++
        merge(&mix[l][dest], higher32(data64), r2)
    }
}

func progpow(hash []byte, nonce uint64, size uint64, blockNumber uint64,
                  lookup func(index uint32) uint32) ([]byte, []byte) {
    var mix [progpowLanes][progpowRegs] uint32
    var laneResults [progpowLanes]      uint32

    cDag   := make([]uint32, progpowCacheWords)
    result := make([]uint32, 8)

    // initialize cDag
    for i := uint32(0); i < progpowCacheWords; i+=2 {
        cDag[i]   = lookup(2*i)
        cDag[i+1] = lookup(2*i+1)
    }
    for i := uint32(0); i < 8; i++ {
        result[i] = 0
    }

    seed := keccakF800Short(hash, nonce, result)
    for lane := uint32(0); lane < progpowLanes; lane++ {
        mix[lane] = fillMix(seed, lane)
    }

    blockNumberRounded := (blockNumber/epochLength)*epochLength
    for l := uint32(0); l < progpowCntMem; l++ {
        progpowLoop(blockNumberRounded, l, &mix, lookup, cDag,
                    uint32(size/progpowMixBytes))
    }

    // Reduce mix data to a single per-lane result
    for lane := uint32(0); lane < progpowLanes; lane++ {
        laneResults[lane] = 0x811c9dc5
        for i := uint32(0); i < progpowRegs; i++ {
            fnv1a(&laneResults[lane], mix[lane][i])
        }
    }

    for i := uint32(0); i < 8; i++ {
        result[i] = 0x811c9dc5
    }
    for lane := uint32(0); lane < progpowLanes; lane++ {
        fnv1a(&result[lane%8], laneResults[lane])
    }

    digest := keccakF800Long(hash, seed, result[:])

    resultBytes := make([]byte, 8*4)
    for i := 0; i < 8; i++ {
        binary.LittleEndian.PutUint32(resultBytes[i*4:], result[i])
    }

    return digest[:], resultBytes[:]
}