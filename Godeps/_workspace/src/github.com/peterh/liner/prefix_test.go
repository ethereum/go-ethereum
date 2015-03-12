// +build windows linux darwin openbsd freebsd netbsd

package liner

import "testing"

type testItem struct {
	list   []string
	prefix string
}

func TestPrefix(t *testing.T) {
	list := []testItem{
		{[]string{"food", "foot"}, "foo"},
		{[]string{"foo", "foot"}, "foo"},
		{[]string{"food", "foo"}, "foo"},
		{[]string{"food", "foe", "foot"}, "fo"},
		{[]string{"food", "foot", "barbeque"}, ""},
		{[]string{"cafeteria", "café"}, "caf"},
		{[]string{"cafe", "café"}, "caf"},
		{[]string{"cafè", "café"}, "caf"},
		{[]string{"cafés", "café"}, "café"},
		{[]string{"áéíóú", "áéíóú"}, "áéíóú"},
		{[]string{"éclairs", "éclairs"}, "éclairs"},
		{[]string{"éclairs are the best", "éclairs are great", "éclairs"}, "éclairs"},
		{[]string{"éclair", "éclairs"}, "éclair"},
		{[]string{"éclairs", "éclair"}, "éclair"},
		{[]string{"éclair", "élan"}, "é"},
	}

	for _, test := range list {
		lcp := longestCommonPrefix(test.list)
		if lcp != test.prefix {
			t.Errorf("%s != %s for %+v", lcp, test.prefix, test.list)
		}
	}
}
