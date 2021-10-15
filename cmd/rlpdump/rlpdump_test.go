package main

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestTextToRlp(t *testing.T) {
	type tc struct {
		text string
		want string
	}
	cases := []tc{
		{
			text: `[
  "",
  "d",
  5208,
  d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0,
  010000000000000000000000000000000000000000000000000000000000000001,
  "",
  1b,
  c16787a8e25e941d67691954642876c08f00996163ae7dfadbbfd6cd436f549d,
  6180e5626cae31590f40641fe8f63734316c4bfeb4cdfab6714198c1044d2e28,
]`,
			want: "0xf880806482520894d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0a1010000000000000000000000000000000000000000000000000000000000000001801ba0c16787a8e25e941d67691954642876c08f00996163ae7dfadbbfd6cd436f549da06180e5626cae31590f40641fe8f63734316c4bfeb4cdfab6714198c1044d2e28",
		},
		{ // Same as above but no ascii
			text: `[
  "",
  64,
  5208,
  d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0,
  010000000000000000000000000000000000000000000000000000000000000001,
  "",
  1b,
  c16787a8e25e941d67691954642876c08f00996163ae7dfadbbfd6cd436f549d,
  6180e5626cae31590f40641fe8f63734316c4bfeb4cdfab6714198c1044d2e28,
]
`,
			want: "0xf880806482520894d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0d0a1010000000000000000000000000000000000000000000000000000000000000001801ba0c16787a8e25e941d67691954642876c08f00996163ae7dfadbbfd6cd436f549da06180e5626cae31590f40641fe8f63734316c4bfeb4cdfab6714198c1044d2e28",
		},
		{
			text: `
[
  [],
  [
    [
      "test",
      "*",
      "*",
      "",
      1337,
    ],
    "gazonk",
  ],
]

`,
			want: "0xd5c0d3cb84746573742a2a808213378667617a6f6e6b",
		},
	}
	for i, tc := range cases {
		have, err := textToRlp(strings.NewReader(tc.text))
		if err != nil {
			t.Errorf("test %d: error %v", i, err)
			continue
		}
		if hexutil.Encode(have) != tc.want {
			t.Errorf("test %d:\nhave %v\nwant %v", i, hexutil.Encode(have), tc.want)
		}
	}
}
