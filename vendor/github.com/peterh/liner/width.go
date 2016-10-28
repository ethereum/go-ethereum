package liner

import "unicode"

// These character classes are mostly zero width (when combined).
// A few might not be, depending on the user's font. Fixing this
// is non-trivial, given that some terminals don't support
// ANSI DSR/CPR
var zeroWidth = []*unicode.RangeTable{
	unicode.Mn,
	unicode.Me,
	unicode.Cc,
	unicode.Cf,
}

var doubleWidth = []*unicode.RangeTable{
	unicode.Han,
	unicode.Hangul,
	unicode.Hiragana,
	unicode.Katakana,
}

// countGlyphs considers zero-width characters to be zero glyphs wide,
// and members of Chinese, Japanese, and Korean scripts to be 2 glyphs wide.
func countGlyphs(s []rune) int {
	n := 0
	for _, r := range s {
		switch {
		case unicode.IsOneOf(zeroWidth, r):
		case unicode.IsOneOf(doubleWidth, r):
			n += 2
		default:
			n++
		}
	}
	return n
}

func countMultiLineGlyphs(s []rune, columns int, start int) int {
	n := start
	for _, r := range s {
		switch {
		case unicode.IsOneOf(zeroWidth, r):
		case unicode.IsOneOf(doubleWidth, r):
			n += 2
			// no room for a 2-glyphs-wide char in the ending
			// so skip a column and display it at the beginning
			if n%columns == 1 {
				n++
			}
		default:
			n++
		}
	}
	return n
}

func getPrefixGlyphs(s []rune, num int) []rune {
	p := 0
	for n := 0; n < num && p < len(s); p++ {
		if !unicode.IsOneOf(zeroWidth, s[p]) {
			n++
		}
	}
	for p < len(s) && unicode.IsOneOf(zeroWidth, s[p]) {
		p++
	}
	return s[:p]
}

func getSuffixGlyphs(s []rune, num int) []rune {
	p := len(s)
	for n := 0; n < num && p > 0; p-- {
		if !unicode.IsOneOf(zeroWidth, s[p-1]) {
			n++
		}
	}
	return s[p:]
}
