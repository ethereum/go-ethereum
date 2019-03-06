package base58

import (
	"fmt"
	"math/big"
)

var (
	bn0  = big.NewInt(0)
	bn58 = big.NewInt(58)
)

// Encode encodes the passed bytes into a base58 encoded string.
func Encode(bin []byte) string {
	return FastBase58Encoding(bin)
}

// EncodeAlphabet encodes the passed bytes into a base58 encoded string with the
// passed alphabet.
func EncodeAlphabet(bin []byte, alphabet *Alphabet) string {
	return FastBase58EncodingAlphabet(bin, alphabet)
}

// FastBase58Encoding encodes the passed bytes into a base58 encoded string.
func FastBase58Encoding(bin []byte) string {
	return FastBase58EncodingAlphabet(bin, BTCAlphabet)
}

// FastBase58EncodingAlphabet encodes the passed bytes into a base58 encoded
// string with the passed alphabet.
func FastBase58EncodingAlphabet(bin []byte, alphabet *Alphabet) string {
	zero := alphabet.encode[0]

	binsz := len(bin)
	var i, j, zcount, high int
	var carry uint32

	for zcount < binsz && bin[zcount] == 0 {
		zcount++
	}

	size := (binsz-zcount)*138/100 + 1
	var buf = make([]byte, size)

	high = size - 1
	for i = zcount; i < binsz; i++ {
		j = size - 1
		for carry = uint32(bin[i]); j > high || carry != 0; j-- {
			carry = carry + 256*uint32(buf[j])
			buf[j] = byte(carry % 58)
			carry /= 58
		}
		high = j
	}

	for j = 0; j < size && buf[j] == 0; j++ {
	}

	var b58 = make([]byte, size-j+zcount)

	if zcount != 0 {
		for i = 0; i < zcount; i++ {
			b58[i] = zero
		}
	}

	for i = zcount; j < size; i++ {
		b58[i] = alphabet.encode[buf[j]]
		j++
	}

	return string(b58)
}

// TrivialBase58Encoding encodes the passed bytes into a base58 encoded string
// (inefficiently).
func TrivialBase58Encoding(a []byte) string {
	return TrivialBase58EncodingAlphabet(a, BTCAlphabet)
}

// TrivialBase58EncodingAlphabet encodes the passed bytes into a base58 encoded
// string (inefficiently) with the passed alphabet.
func TrivialBase58EncodingAlphabet(a []byte, alphabet *Alphabet) string {
	zero := alphabet.encode[0]
	idx := len(a)*138/100 + 1
	buf := make([]byte, idx)
	bn := new(big.Int).SetBytes(a)
	var mo *big.Int
	for bn.Cmp(bn0) != 0 {
		bn, mo = bn.DivMod(bn, bn58, new(big.Int))
		idx--
		buf[idx] = alphabet.encode[mo.Int64()]
	}
	for i := range a {
		if a[i] != 0 {
			break
		}
		idx--
		buf[idx] = zero
	}
	return string(buf[idx:])
}

// Decode decodes the base58 encoded bytes.
func Decode(str string) ([]byte, error) {
	return FastBase58Decoding(str)
}

// DecodeAlphabet decodes the base58 encoded bytes using the given b58 alphabet.
func DecodeAlphabet(str string, alphabet *Alphabet) ([]byte, error) {
	return FastBase58DecodingAlphabet(str, alphabet)
}

// FastBase58Decoding decodes the base58 encoded bytes.
func FastBase58Decoding(str string) ([]byte, error) {
	return FastBase58DecodingAlphabet(str, BTCAlphabet)
}

// FastBase58DecodingAlphabet decodes the base58 encoded bytes using the given
// b58 alphabet.
func FastBase58DecodingAlphabet(str string, alphabet *Alphabet) ([]byte, error) {
	if len(str) == 0 {
		return nil, fmt.Errorf("zero length string")
	}

	var (
		t        uint64
		zmask, c uint32
		zcount   int

		b58u  = []rune(str)
		b58sz = len(b58u)

		outisz    = (b58sz + 3) / 4 // check to see if we need to change this buffer size to optimize
		binu      = make([]byte, (b58sz+3)*3)
		bytesleft = b58sz % 4

		zero = rune(alphabet.encode[0])
	)

	if bytesleft > 0 {
		zmask = (0xffffffff << uint32(bytesleft*8))
	} else {
		bytesleft = 4
	}

	var outi = make([]uint32, outisz)

	for i := 0; i < b58sz && b58u[i] == zero; i++ {
		zcount++
	}

	for _, r := range b58u {
		if r > 127 {
			return nil, fmt.Errorf("High-bit set on invalid digit")
		}
		if alphabet.decode[r] == -1 {
			return nil, fmt.Errorf("Invalid base58 digit (%q)", r)
		}

		c = uint32(alphabet.decode[r])

		for j := (outisz - 1); j >= 0; j-- {
			t = uint64(outi[j])*58 + uint64(c)
			c = uint32(t>>32) & 0x3f
			outi[j] = uint32(t & 0xffffffff)
		}

		if c > 0 {
			return nil, fmt.Errorf("Output number too big (carry to the next int32)")
		}

		if outi[0]&zmask != 0 {
			return nil, fmt.Errorf("Output number too big (last int32 filled too far)")
		}
	}

	// the nested for-loop below is the same as the original code:
	// switch (bytesleft) {
	// 	case 3:
	// 		*(binu++) = (outi[0] & 0xff0000) >> 16;
	// 		//-fallthrough
	// 	case 2:
	// 		*(binu++) = (outi[0] & 0xff00) >>  8;
	// 		//-fallthrough
	// 	case 1:
	// 		*(binu++) = (outi[0] & 0xff);
	// 		++j;
	// 		//-fallthrough
	// 	default:
	// 		break;
	// }
	//
	// for (; j < outisz; ++j)
	// {
	// 	*(binu++) = (outi[j] >> 0x18) & 0xff;
	// 	*(binu++) = (outi[j] >> 0x10) & 0xff;
	// 	*(binu++) = (outi[j] >>    8) & 0xff;
	// 	*(binu++) = (outi[j] >>    0) & 0xff;
	// }
	var j, cnt int
	for j, cnt = 0, 0; j < outisz; j++ {
		for mask := byte(bytesleft-1) * 8; mask <= 0x18; mask, cnt = mask-8, cnt+1 {
			binu[cnt] = byte(outi[j] >> mask)
		}
		if j == 0 {
			bytesleft = 4 // because it could be less than 4 the first time through
		}
	}

	for n, v := range binu {
		if v > 0 {
			start := n - zcount
			if start < 0 {
				start = 0
			}
			return binu[start:cnt], nil
		}
	}
	return binu[:cnt], nil
}

// TrivialBase58Decoding decodes the base58 encoded bytes (inefficiently).
func TrivialBase58Decoding(str string) ([]byte, error) {
	return TrivialBase58DecodingAlphabet(str, BTCAlphabet)
}

// TrivialBase58DecodingAlphabet decodes the base58 encoded bytes
// (inefficiently) using the given b58 alphabet.
func TrivialBase58DecodingAlphabet(str string, alphabet *Alphabet) ([]byte, error) {
	zero := alphabet.encode[0]

	var zcount int
	for i := 0; i < len(str) && str[i] == zero; i++ {
		zcount++
	}
	leading := make([]byte, zcount)

	var padChar rune = -1
	src := []byte(str)
	j := 0
	for ; j < len(src) && src[j] == byte(padChar); j++ {
	}

	n := new(big.Int)
	for i := range src[j:] {
		c := alphabet.decode[src[i]]
		if c == -1 {
			return nil, fmt.Errorf("illegal base58 data at input index: %d", i)
		}
		n.Mul(n, bn58)
		n.Add(n, big.NewInt(int64(c)))
	}
	return append(leading, n.Bytes()...), nil
}
