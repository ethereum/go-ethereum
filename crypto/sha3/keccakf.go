// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sha3

// This file implements the core Keccak permutation function necessary for computing SHA3.
// This is implemented in a separate file to allow for replacement by an optimized implementation.
// Nothing in this package is exported.
// For the detailed specification, refer to the Keccak web site (http://keccak.noekeon.org/).

// rc stores the round constants for use in the ι step.
var rc = [...]uint64{
        0x0000000000000001,
        0x0000000000008082,
        0x800000000000808A,
        0x8000000080008000,
        0x000000000000808B,
        0x0000000080000001,
        0x8000000080008081,
        0x8000000000008009,
        0x000000000000008A,
        0x0000000000000088,
        0x0000000080008009,
        0x000000008000000A,
        0x000000008000808B,
        0x800000000000008B,
        0x8000000000008089,
        0x8000000000008003,
        0x8000000000008002,
        0x8000000000000080,
        0x000000000000800A,
        0x800000008000000A,
        0x8000000080008081,
        0x8000000000008080,
        0x0000000080000001,
        0x8000000080008008,
}

// ro_xx represent the rotation offsets for use in the χ step.
// Defining them as const instead of in an array allows the compiler to insert constant shifts.
const (
        ro_00 = 0
        ro_01 = 36
        ro_02 = 3
        ro_03 = 41
        ro_04 = 18
        ro_05 = 1
        ro_06 = 44
        ro_07 = 10
        ro_08 = 45
        ro_09 = 2
        ro_10 = 62
        ro_11 = 6
        ro_12 = 43
        ro_13 = 15
        ro_14 = 61
        ro_15 = 28
        ro_16 = 55
        ro_17 = 25
        ro_18 = 21
        ro_19 = 56
        ro_20 = 27
        ro_21 = 20
        ro_22 = 39
        ro_23 = 8
        ro_24 = 14
)

// keccakF computes the complete Keccak-f function consisting of 24 rounds with a different
// constant (rc) in each round. This implementation fully unrolls the round function to avoid
// inner loops, as well as pre-calculating shift offsets.
func (d *digest) keccakF() {
        for _, roundConstant := range rc {
                // θ step
                d.c[0] = d.a[0] ^ d.a[5] ^ d.a[10] ^ d.a[15] ^ d.a[20]
                d.c[1] = d.a[1] ^ d.a[6] ^ d.a[11] ^ d.a[16] ^ d.a[21]
                d.c[2] = d.a[2] ^ d.a[7] ^ d.a[12] ^ d.a[17] ^ d.a[22]
                d.c[3] = d.a[3] ^ d.a[8] ^ d.a[13] ^ d.a[18] ^ d.a[23]
                d.c[4] = d.a[4] ^ d.a[9] ^ d.a[14] ^ d.a[19] ^ d.a[24]

                d.d[0] = d.c[4] ^ (d.c[1]<<1 ^ d.c[1]>>63)
                d.d[1] = d.c[0] ^ (d.c[2]<<1 ^ d.c[2]>>63)
                d.d[2] = d.c[1] ^ (d.c[3]<<1 ^ d.c[3]>>63)
                d.d[3] = d.c[2] ^ (d.c[4]<<1 ^ d.c[4]>>63)
                d.d[4] = d.c[3] ^ (d.c[0]<<1 ^ d.c[0]>>63)

                d.a[0] ^= d.d[0]
                d.a[1] ^= d.d[1]
                d.a[2] ^= d.d[2]
                d.a[3] ^= d.d[3]
                d.a[4] ^= d.d[4]
                d.a[5] ^= d.d[0]
                d.a[6] ^= d.d[1]
                d.a[7] ^= d.d[2]
                d.a[8] ^= d.d[3]
                d.a[9] ^= d.d[4]
                d.a[10] ^= d.d[0]
                d.a[11] ^= d.d[1]
                d.a[12] ^= d.d[2]
                d.a[13] ^= d.d[3]
                d.a[14] ^= d.d[4]
                d.a[15] ^= d.d[0]
                d.a[16] ^= d.d[1]
                d.a[17] ^= d.d[2]
                d.a[18] ^= d.d[3]
                d.a[19] ^= d.d[4]
                d.a[20] ^= d.d[0]
                d.a[21] ^= d.d[1]
                d.a[22] ^= d.d[2]
                d.a[23] ^= d.d[3]
                d.a[24] ^= d.d[4]

                // ρ and π steps
                d.b[0] = d.a[0]
                d.b[1] = d.a[6]<<ro_06 ^ d.a[6]>>(64-ro_06)
                d.b[2] = d.a[12]<<ro_12 ^ d.a[12]>>(64-ro_12)
                d.b[3] = d.a[18]<<ro_18 ^ d.a[18]>>(64-ro_18)
                d.b[4] = d.a[24]<<ro_24 ^ d.a[24]>>(64-ro_24)
                d.b[5] = d.a[3]<<ro_15 ^ d.a[3]>>(64-ro_15)
                d.b[6] = d.a[9]<<ro_21 ^ d.a[9]>>(64-ro_21)
                d.b[7] = d.a[10]<<ro_02 ^ d.a[10]>>(64-ro_02)
                d.b[8] = d.a[16]<<ro_08 ^ d.a[16]>>(64-ro_08)
                d.b[9] = d.a[22]<<ro_14 ^ d.a[22]>>(64-ro_14)
                d.b[10] = d.a[1]<<ro_05 ^ d.a[1]>>(64-ro_05)
                d.b[11] = d.a[7]<<ro_11 ^ d.a[7]>>(64-ro_11)
                d.b[12] = d.a[13]<<ro_17 ^ d.a[13]>>(64-ro_17)
                d.b[13] = d.a[19]<<ro_23 ^ d.a[19]>>(64-ro_23)
                d.b[14] = d.a[20]<<ro_04 ^ d.a[20]>>(64-ro_04)
                d.b[15] = d.a[4]<<ro_20 ^ d.a[4]>>(64-ro_20)
                d.b[16] = d.a[5]<<ro_01 ^ d.a[5]>>(64-ro_01)
                d.b[17] = d.a[11]<<ro_07 ^ d.a[11]>>(64-ro_07)
                d.b[18] = d.a[17]<<ro_13 ^ d.a[17]>>(64-ro_13)
                d.b[19] = d.a[23]<<ro_19 ^ d.a[23]>>(64-ro_19)
                d.b[20] = d.a[2]<<ro_10 ^ d.a[2]>>(64-ro_10)
                d.b[21] = d.a[8]<<ro_16 ^ d.a[8]>>(64-ro_16)
                d.b[22] = d.a[14]<<ro_22 ^ d.a[14]>>(64-ro_22)
                d.b[23] = d.a[15]<<ro_03 ^ d.a[15]>>(64-ro_03)
                d.b[24] = d.a[21]<<ro_09 ^ d.a[21]>>(64-ro_09)

                // χ step
                d.a[0] = d.b[0] ^ (^d.b[1] & d.b[2])
                d.a[1] = d.b[1] ^ (^d.b[2] & d.b[3])
                d.a[2] = d.b[2] ^ (^d.b[3] & d.b[4])
                d.a[3] = d.b[3] ^ (^d.b[4] & d.b[0])
                d.a[4] = d.b[4] ^ (^d.b[0] & d.b[1])
                d.a[5] = d.b[5] ^ (^d.b[6] & d.b[7])
                d.a[6] = d.b[6] ^ (^d.b[7] & d.b[8])
                d.a[7] = d.b[7] ^ (^d.b[8] & d.b[9])
                d.a[8] = d.b[8] ^ (^d.b[9] & d.b[5])
                d.a[9] = d.b[9] ^ (^d.b[5] & d.b[6])
                d.a[10] = d.b[10] ^ (^d.b[11] & d.b[12])
                d.a[11] = d.b[11] ^ (^d.b[12] & d.b[13])
                d.a[12] = d.b[12] ^ (^d.b[13] & d.b[14])
                d.a[13] = d.b[13] ^ (^d.b[14] & d.b[10])
                d.a[14] = d.b[14] ^ (^d.b[10] & d.b[11])
                d.a[15] = d.b[15] ^ (^d.b[16] & d.b[17])
                d.a[16] = d.b[16] ^ (^d.b[17] & d.b[18])
                d.a[17] = d.b[17] ^ (^d.b[18] & d.b[19])
                d.a[18] = d.b[18] ^ (^d.b[19] & d.b[15])
                d.a[19] = d.b[19] ^ (^d.b[15] & d.b[16])
                d.a[20] = d.b[20] ^ (^d.b[21] & d.b[22])
                d.a[21] = d.b[21] ^ (^d.b[22] & d.b[23])
                d.a[22] = d.b[22] ^ (^d.b[23] & d.b[24])
                d.a[23] = d.b[23] ^ (^d.b[24] & d.b[20])
                d.a[24] = d.b[24] ^ (^d.b[20] & d.b[21])

                // ι step
                d.a[0] ^= roundConstant
        }
}
