package liner

import (
	"strconv"
	"testing"
)

func accent(in []rune) []rune {
	var out []rune
	for _, r := range in {
		out = append(out, r)
		out = append(out, '\u0301')
	}
	return out
}

var testString = []rune("query")

func TestCountGlyphs(t *testing.T) {
	count := countGlyphs(testString)
	if count != len(testString) {
		t.Errorf("ASCII count incorrect. %d != %d", count, len(testString))
	}
	count = countGlyphs(accent(testString))
	if count != len(testString) {
		t.Errorf("Accent count incorrect. %d != %d", count, len(testString))
	}
}

func compare(a, b []rune, name string, t *testing.T) {
	if len(a) != len(b) {
		t.Errorf(`"%s" != "%s" in %s"`, string(a), string(b), name)
		return
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf(`"%s" != "%s" in %s"`, string(a), string(b), name)
			return
		}
	}
}

func TestPrefixGlyphs(t *testing.T) {
	for i := 0; i <= len(testString); i++ {
		iter := strconv.Itoa(i)
		out := getPrefixGlyphs(testString, i)
		compare(out, testString[:i], "ascii prefix "+iter, t)
		out = getPrefixGlyphs(accent(testString), i)
		compare(out, accent(testString[:i]), "accent prefix "+iter, t)
	}
	out := getPrefixGlyphs(testString, 999)
	compare(out, testString, "ascii prefix overflow", t)
	out = getPrefixGlyphs(accent(testString), 999)
	compare(out, accent(testString), "accent prefix overflow", t)

	out = getPrefixGlyphs(testString, -3)
	if len(out) != 0 {
		t.Error("ascii prefix negative")
	}
	out = getPrefixGlyphs(accent(testString), -3)
	if len(out) != 0 {
		t.Error("accent prefix negative")
	}
}

func TestSuffixGlyphs(t *testing.T) {
	for i := 0; i <= len(testString); i++ {
		iter := strconv.Itoa(i)
		out := getSuffixGlyphs(testString, i)
		compare(out, testString[len(testString)-i:], "ascii suffix "+iter, t)
		out = getSuffixGlyphs(accent(testString), i)
		compare(out, accent(testString[len(testString)-i:]), "accent suffix "+iter, t)
	}
	out := getSuffixGlyphs(testString, 999)
	compare(out, testString, "ascii suffix overflow", t)
	out = getSuffixGlyphs(accent(testString), 999)
	compare(out, accent(testString), "accent suffix overflow", t)

	out = getSuffixGlyphs(testString, -3)
	if len(out) != 0 {
		t.Error("ascii suffix negative")
	}
	out = getSuffixGlyphs(accent(testString), -3)
	if len(out) != 0 {
		t.Error("accent suffix negative")
	}
}
