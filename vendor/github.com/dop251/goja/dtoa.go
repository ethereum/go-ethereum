package goja

// Ported from Rhino (https://github.com/mozilla/rhino/blob/master/src/org/mozilla/javascript/DToA.java)

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

const (
	frac_mask = 0xfffff
	exp_shift = 20
	exp_msk1  = 0x100000

	exp_shiftL       = 52
	exp_mask_shifted = 0x7ff
	frac_maskL       = 0xfffffffffffff
	exp_msk1L        = 0x10000000000000
	exp_shift1       = 20
	exp_mask         = 0x7ff00000
	bias             = 1023
	p                = 53
	bndry_mask       = 0xfffff
	log2P            = 1

	digits = "0123456789abcdefghijklmnopqrstuvwxyz"
)

func lo0bits(x uint32) (k uint32) {

	if (x & 7) != 0 {
		if (x & 1) != 0 {
			return 0
		}
		if (x & 2) != 0 {
			return 1
		}
		return 2
	}
	if (x & 0xffff) == 0 {
		k = 16
		x >>= 16
	}
	if (x & 0xff) == 0 {
		k += 8
		x >>= 8
	}
	if (x & 0xf) == 0 {
		k += 4
		x >>= 4
	}
	if (x & 0x3) == 0 {
		k += 2
		x >>= 2
	}
	if (x & 1) == 0 {
		k++
		x >>= 1
		if (x & 1) == 0 {
			return 32
		}
	}
	return
}

func hi0bits(x uint32) (k uint32) {

	if (x & 0xffff0000) == 0 {
		k = 16
		x <<= 16
	}
	if (x & 0xff000000) == 0 {
		k += 8
		x <<= 8
	}
	if (x & 0xf0000000) == 0 {
		k += 4
		x <<= 4
	}
	if (x & 0xc0000000) == 0 {
		k += 2
		x <<= 2
	}
	if (x & 0x80000000) == 0 {
		k++
		if (x & 0x40000000) == 0 {
			return 32
		}
	}
	return
}

func stuffBits(bits []byte, offset int, val uint32) {
	bits[offset] = byte(val >> 24)
	bits[offset+1] = byte(val >> 16)
	bits[offset+2] = byte(val >> 8)
	bits[offset+3] = byte(val)
}

func d2b(d float64) (b *big.Int, e int32, bits uint32) {
	dBits := math.Float64bits(d)
	d0 := uint32(dBits >> 32)
	d1 := uint32(dBits)

	z := d0 & frac_mask
	d0 &= 0x7fffffff /* clear sign bit, which we ignore */

	var de, k, i uint32
	var dbl_bits []byte
	if de = (d0 >> exp_shift); de != 0 {
		z |= exp_msk1
	}

	y := d1
	if y != 0 {
		dbl_bits = make([]byte, 8)
		k = lo0bits(y)
		y >>= k
		if k != 0 {
			stuffBits(dbl_bits, 4, y|z<<(32-k))
			z >>= k
		} else {
			stuffBits(dbl_bits, 4, y)
		}
		stuffBits(dbl_bits, 0, z)
		if z != 0 {
			i = 2
		} else {
			i = 1
		}
	} else {
		dbl_bits = make([]byte, 4)
		k = lo0bits(z)
		z >>= k
		stuffBits(dbl_bits, 0, z)
		k += 32
		i = 1
	}

	if de != 0 {
		e = int32(de - bias - (p - 1) + k)
		bits = p - k
	} else {
		e = int32(de - bias - (p - 1) + 1 + k)
		bits = 32*i - hi0bits(z)
	}
	b = (&big.Int{}).SetBytes(dbl_bits)
	return
}

func dtobasestr(num float64, radix int) string {
	var negative bool
	if num < 0 {
		num = -num
		negative = true
	}

	dfloor := math.Floor(num)
	ldfloor := int64(dfloor)
	var intDigits string
	if dfloor == float64(ldfloor) {
		if negative {
			ldfloor = -ldfloor
		}
		intDigits = strconv.FormatInt(ldfloor, radix)
	} else {
		floorBits := math.Float64bits(num)
		exp := int(floorBits>>exp_shiftL) & exp_mask_shifted
		var mantissa int64
		if exp == 0 {
			mantissa = int64((floorBits & frac_maskL) << 1)
		} else {
			mantissa = int64((floorBits & frac_maskL) | exp_msk1L)
		}

		if negative {
			mantissa = -mantissa
		}
		exp -= 1075
		x := big.NewInt(mantissa)
		if exp > 0 {
			x.Lsh(x, uint(exp))
		} else if exp < 0 {
			x.Rsh(x, uint(-exp))
		}
		intDigits = x.Text(radix)
	}

	if num == dfloor {
		// No fraction part
		return intDigits
	} else {
		/* We have a fraction. */
		var buffer bytes.Buffer
		buffer.WriteString(intDigits)
		buffer.WriteByte('.')
		df := num - dfloor

		dBits := math.Float64bits(num)
		word0 := uint32(dBits >> 32)
		word1 := uint32(dBits)

		b, e, _ := d2b(df)
		//            JS_ASSERT(e < 0);
		/* At this point df = b * 2^e.  e must be less than zero because 0 < df < 1. */

		s2 := -int32((word0 >> exp_shift1) & (exp_mask >> exp_shift1))
		if s2 == 0 {
			s2 = -1
		}
		s2 += bias + p
		/* 1/2^s2 = (nextDouble(d) - d)/2 */
		//            JS_ASSERT(-s2 < e);
		if -s2 >= e {
			panic(fmt.Errorf("-s2 >= e: %d, %d", -s2, e))
		}
		mlo := big.NewInt(1)
		mhi := mlo
		if (word1 == 0) && ((word0 & bndry_mask) == 0) && ((word0 & (exp_mask & (exp_mask << 1))) != 0) {
			/* The special case.  Here we want to be within a quarter of the last input
			   significant digit instead of one half of it when the output string's value is less than d.  */
			s2 += log2P
			mhi = big.NewInt(1 << log2P)
		}

		b.Lsh(b, uint(e+s2))
		s := big.NewInt(1)
		s.Lsh(s, uint(s2))
		/* At this point we have the following:
		 *   s = 2^s2;
		 *   1 > df = b/2^s2 > 0;
		 *   (d - prevDouble(d))/2 = mlo/2^s2;
		 *   (nextDouble(d) - d)/2 = mhi/2^s2. */
		bigBase := big.NewInt(int64(radix))

		done := false
		m := &big.Int{}
		delta := &big.Int{}
		for !done {
			b.Mul(b, bigBase)
			b.DivMod(b, s, m)
			digit := byte(b.Int64())
			b, m = m, b
			mlo.Mul(mlo, bigBase)
			if mlo != mhi {
				mhi.Mul(mhi, bigBase)
			}

			/* Do we yet have the shortest string that will round to d? */
			j := b.Cmp(mlo)
			/* j is b/2^s2 compared with mlo/2^s2. */

			delta.Sub(s, mhi)
			var j1 int
			if delta.Sign() <= 0 {
				j1 = 1
			} else {
				j1 = b.Cmp(delta)
			}
			/* j1 is b/2^s2 compared with 1 - mhi/2^s2. */
			if j1 == 0 && (word1&1) == 0 {
				if j > 0 {
					digit++
				}
				done = true
			} else if j < 0 || (j == 0 && ((word1 & 1) == 0)) {
				if j1 > 0 {
					/* Either dig or dig+1 would work here as the least significant digit.
					Use whichever would produce an output value closer to d. */
					b.Lsh(b, 1)
					j1 = b.Cmp(s)
					if j1 > 0 { /* The even test (|| (j1 == 0 && (digit & 1))) is not here because it messes up odd base output such as 3.5 in base 3.  */
						digit++
					}
				}
				done = true
			} else if j1 > 0 {
				digit++
				done = true
			}
			//                JS_ASSERT(digit < (uint32)base);
			buffer.WriteByte(digits[digit])
		}

		return buffer.String()
	}
}
