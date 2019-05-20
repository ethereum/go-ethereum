// Copyright 2015 Zack Guo <gizak@icloud.com>. All rights reserved.
// Use of this source code is governed by a MIT license that can
// be found in the LICENSE file.

package termui

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestStr2Rune(t *testing.T) {
	s := "你好,世界."
	rs := str2runes(s)
	if len(rs) != 6 {
		t.Error()
	}
}

func TestWidth(t *testing.T) {
	s0 := "つのだ☆HIRO"
	s1 := "11111111111"
	spew.Dump(s0)
	spew.Dump(s1)
	// above not align for setting East Asian Ambiguous to wide!!

	if strWidth(s0) != strWidth(s1) {
		t.Error("str len failed")
	}

	len1 := []rune{'a', '2', '&', '｢', 'ｵ', '｡'} //will false: 'ᆵ', 'ᄚ', 'ᄒ'
	for _, v := range len1 {
		if charWidth(v) != 1 {
			t.Error("len1 failed")
		}
	}

	len2 := []rune{'漢', '字', '한', '자', '你', '好', 'だ', '。', '％', 'ｓ', 'Ｅ', 'ョ', '、', 'ヲ'}
	for _, v := range len2 {
		if charWidth(v) != 2 {
			t.Error("len2 failed")
		}
	}
}

func TestTrim(t *testing.T) {
	s := "つのだ☆HIRO"
	if string(trimStr2Runes(s, 10)) != "つのだ☆HI"+dot {
		t.Error("trim failed")
	}
	if string(trimStr2Runes(s, 11)) != "つのだ☆HIRO" {
		t.Error("avoid tail trim failed")
	}
	if string(trimStr2Runes(s, 15)) != "つのだ☆HIRO" {
		t.Error("avoid trim failed")
	}
}
