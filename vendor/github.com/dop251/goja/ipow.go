package goja

// ported from https://gist.github.com/orlp/3551590

var highest_bit_set = [256]byte{
	0, 1, 2, 2, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 4,
	5, 5, 5, 5, 5, 5, 5, 5,
	5, 5, 5, 5, 5, 5, 5, 5,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 6,
	6, 6, 6, 6, 6, 6, 6, 255, // anything past 63 is a guaranteed overflow with base > 1
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255,
}

func ipow(base, exp int64) (result int64) {
	result = 1

	switch highest_bit_set[byte(exp)] {
	case 255: // we use 255 as an overflow marker and return 0 on overflow/underflow
		if base == 1 {
			return 1
		}

		if base == -1 {
			return 1 - 2*(exp&1)
		}

		return 0
	case 6:
		if exp&1 != 0 {
			result *= base
		}
		exp >>= 1
		base *= base
		fallthrough
	case 5:
		if exp&1 != 0 {
			result *= base
		}
		exp >>= 1
		base *= base
		fallthrough
	case 4:
		if exp&1 != 0 {
			result *= base
		}
		exp >>= 1
		base *= base
		fallthrough
	case 3:
		if exp&1 != 0 {
			result *= base
		}
		exp >>= 1
		base *= base
		fallthrough
	case 2:
		if exp&1 != 0 {
			result *= base
		}
		exp >>= 1
		base *= base
		fallthrough
	case 1:
		if exp&1 != 0 {
			result *= base
		}
		fallthrough
	default:
		return result
	}
}
