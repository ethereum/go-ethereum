package cors

import (
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	s := convert([]string{"A", "b", "C"}, strings.ToLower)
	e := []string{"a", "b", "c"}
	if s[0] != e[0] || s[1] != e[1] || s[2] != e[2] {
		t.Errorf("%v != %v", s, e)
	}
}

func TestParseHeaderList(t *testing.T) {
	h := parseHeaderList("header, second-header, THIRD-HEADER")
	e := []string{"Header", "Second-Header", "Third-Header"}
	if h[0] != e[0] || h[1] != e[1] || h[2] != e[2] {
		t.Errorf("%v != %v", h, e)
	}
}

func TestParseHeaderListEmpty(t *testing.T) {
	if len(parseHeaderList("")) != 0 {
		t.Error("should be empty sclice")
	}
}
