// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

// This program generates tables.go:
//	go run maketables.go | gofmt > tables.go

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
)

const ascii = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	` !"#$%&'()*+,-./0123456789:;<=>?` +
	`@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\]^_` +
	"`abcdefghijklmnopqrstuvwxyz{|}~\u007f"

var encodings = []struct {
	name        string
	mib         string
	comment     string
	varName     string
	replacement byte
	mapping     string
}{
	{
		"IBM Code Page 437",
		"PC8CodePage437",
		"",
		"CodePage437",
		encoding.ASCIISub,
		ascii +
			"ÇüéâäàåçêëèïîìÄÅÉæÆôöòûùÿÖÜ¢£¥₧ƒ" +
			"áíóúñÑªº¿⌐¬½¼¡«»░▒▓│┤╡╢╖╕╣║╗╝╜╛┐" +
			"└┴┬├─┼╞╟╚╔╩╦╠═╬╧╨╤╥╙╘╒╓╫╪┘┌█▄▌▐▀" +
			"αßΓπΣσµτΦΘΩδ∞∅∈∩≡±≥≤⌠⌡÷≈°•·√ⁿ²∎\u00a0",
	},
	{
		"IBM Code Page 866",
		"IBM866",
		"",
		"CodePage866",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-ibm866.txt",
	},
	{
		"ISO 8859-2",
		"ISOLatin2",
		"",
		"ISO8859_2",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-2.txt",
	},
	{
		"ISO 8859-3",
		"ISOLatin3",
		"",
		"ISO8859_3",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-3.txt",
	},
	{
		"ISO 8859-4",
		"ISOLatin4",
		"",
		"ISO8859_4",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-4.txt",
	},
	{
		"ISO 8859-5",
		"ISOLatinCyrillic",
		"",
		"ISO8859_5",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-5.txt",
	},
	{
		"ISO 8859-6",
		"ISOLatinArabic",
		"",
		"ISO8859_6,ISO8859_6E,ISO8859_6I",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-6.txt",
	},
	{
		"ISO 8859-7",
		"ISOLatinGreek",
		"",
		"ISO8859_7",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-7.txt",
	},
	{
		"ISO 8859-8",
		"ISOLatinHebrew",
		"",
		"ISO8859_8,ISO8859_8E,ISO8859_8I",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-8.txt",
	},
	{
		"ISO 8859-10",
		"ISOLatin6",
		"",
		"ISO8859_10",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-10.txt",
	},
	{
		"ISO 8859-13",
		"ISO885913",
		"",
		"ISO8859_13",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-13.txt",
	},
	{
		"ISO 8859-14",
		"ISO885914",
		"",
		"ISO8859_14",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-14.txt",
	},
	{
		"ISO 8859-15",
		"ISO885915",
		"",
		"ISO8859_15",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-15.txt",
	},
	{
		"ISO 8859-16",
		"ISO885916",
		"",
		"ISO8859_16",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-iso-8859-16.txt",
	},
	{
		"KOI8-R",
		"KOI8R",
		"",
		"KOI8R",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-koi8-r.txt",
	},
	{
		"KOI8-U",
		"KOI8U",
		"",
		"KOI8U",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-koi8-u.txt",
	},
	{
		"Macintosh",
		"Macintosh",
		"",
		"Macintosh",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-macintosh.txt",
	},
	{
		"Macintosh Cyrillic",
		"MacintoshCyrillic",
		"",
		"MacintoshCyrillic",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-x-mac-cyrillic.txt",
	},
	{
		"Windows 874",
		"Windows874",
		"",
		"Windows874",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-874.txt",
	},
	{
		"Windows 1250",
		"Windows1250",
		"",
		"Windows1250",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1250.txt",
	},
	{
		"Windows 1251",
		"Windows1251",
		"",
		"Windows1251",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1251.txt",
	},
	{
		"Windows 1252",
		"Windows1252",
		"",
		"Windows1252",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1252.txt",
	},
	{
		"Windows 1253",
		"Windows1253",
		"",
		"Windows1253",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1253.txt",
	},
	{
		"Windows 1254",
		"Windows1254",
		"",
		"Windows1254",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1254.txt",
	},
	{
		"Windows 1255",
		"Windows1255",
		"",
		"Windows1255",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1255.txt",
	},
	{
		"Windows 1256",
		"Windows1256",
		"",
		"Windows1256",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1256.txt",
	},
	{
		"Windows 1257",
		"Windows1257",
		"",
		"Windows1257",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1257.txt",
	},
	{
		"Windows 1258",
		"Windows1258",
		"",
		"Windows1258",
		encoding.ASCIISub,
		"http://encoding.spec.whatwg.org/index-windows-1258.txt",
	},
	{
		"X-User-Defined",
		"XUserDefined",
		"It is defined at http://encoding.spec.whatwg.org/#x-user-defined",
		"XUserDefined",
		encoding.ASCIISub,
		ascii +
			"\uf780\uf781\uf782\uf783\uf784\uf785\uf786\uf787" +
			"\uf788\uf789\uf78a\uf78b\uf78c\uf78d\uf78e\uf78f" +
			"\uf790\uf791\uf792\uf793\uf794\uf795\uf796\uf797" +
			"\uf798\uf799\uf79a\uf79b\uf79c\uf79d\uf79e\uf79f" +
			"\uf7a0\uf7a1\uf7a2\uf7a3\uf7a4\uf7a5\uf7a6\uf7a7" +
			"\uf7a8\uf7a9\uf7aa\uf7ab\uf7ac\uf7ad\uf7ae\uf7af" +
			"\uf7b0\uf7b1\uf7b2\uf7b3\uf7b4\uf7b5\uf7b6\uf7b7" +
			"\uf7b8\uf7b9\uf7ba\uf7bb\uf7bc\uf7bd\uf7be\uf7bf" +
			"\uf7c0\uf7c1\uf7c2\uf7c3\uf7c4\uf7c5\uf7c6\uf7c7" +
			"\uf7c8\uf7c9\uf7ca\uf7cb\uf7cc\uf7cd\uf7ce\uf7cf" +
			"\uf7d0\uf7d1\uf7d2\uf7d3\uf7d4\uf7d5\uf7d6\uf7d7" +
			"\uf7d8\uf7d9\uf7da\uf7db\uf7dc\uf7dd\uf7de\uf7df" +
			"\uf7e0\uf7e1\uf7e2\uf7e3\uf7e4\uf7e5\uf7e6\uf7e7" +
			"\uf7e8\uf7e9\uf7ea\uf7eb\uf7ec\uf7ed\uf7ee\uf7ef" +
			"\uf7f0\uf7f1\uf7f2\uf7f3\uf7f4\uf7f5\uf7f6\uf7f7" +
			"\uf7f8\uf7f9\uf7fa\uf7fb\uf7fc\uf7fd\uf7fe\uf7ff",
	},
}

func getWHATWG(url string) string {
	res, err := http.Get(url)
	if err != nil {
		log.Fatalf("%q: Get: %v", url, err)
	}
	defer res.Body.Close()

	mapping := make([]rune, 128)
	for i := range mapping {
		mapping[i] = '\ufffd'
	}

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s == "" || s[0] == '#' {
			continue
		}
		x, y := 0, 0
		if _, err := fmt.Sscanf(s, "%d\t0x%x", &x, &y); err != nil {
			log.Fatalf("could not parse %q", s)
		}
		if x < 0 || 128 <= x {
			log.Fatalf("code %d is out of range", x)
		}
		if 0x80 <= y && y < 0xa0 {
			// We diverge from the WHATWG spec by mapping control characters
			// in the range [0x80, 0xa0) to U+FFFD.
			continue
		}
		mapping[x] = rune(y)
	}
	return ascii + string(mapping)
}

func main() {
	mibs := map[string]bool{}
	all := []string{}

	buf := make([]byte, 8)
	fmt.Printf("// generated by go run maketables.go; DO NOT EDIT\n\n")
	fmt.Printf("package charmap\n\n")
	fmt.Printf("import (\n")
	fmt.Printf("\t\"golang.org/x/text/encoding\"\n")
	fmt.Printf("\t\"golang.org/x/text/encoding/internal/identifier\"\n")
	fmt.Printf(")\n\n")
	for _, e := range encodings {
		varNames := strings.Split(e.varName, ",")
		all = append(all, varNames...)
		varName := varNames[0]
		if strings.HasPrefix(e.mapping, "http://encoding.spec.whatwg.org/") {
			e.mapping = getWHATWG(e.mapping)
		}

		asciiSuperset, low := strings.HasPrefix(e.mapping, ascii), 0x00
		if asciiSuperset {
			low = 0x80
		}
		lvn := 1
		if strings.HasPrefix(varName, "ISO") || strings.HasPrefix(varName, "KOI") {
			lvn = 3
		}
		lowerVarName := strings.ToLower(varName[:lvn]) + varName[lvn:]
		fmt.Printf("// %s is the %s encoding.\n", varName, e.name)
		if e.comment != "" {
			fmt.Printf("//\n// %s\n", e.comment)
		}
		fmt.Printf("var %s encoding.Encoding = &%s\n\nvar %s = charmap{\nname: %q,\n",
			varName, lowerVarName, lowerVarName, e.name)
		if mibs[e.mib] {
			log.Fatalf("MIB type %q declared multiple times.", e.mib)
		}
		fmt.Printf("mib: identifier.%s,\n", e.mib)
		fmt.Printf("asciiSuperset: %t,\n", asciiSuperset)
		fmt.Printf("low: 0x%02x,\n", low)
		fmt.Printf("replacement: 0x%02x,\n", e.replacement)

		fmt.Printf("decode: [256]utf8Enc{\n")
		i, backMapping := 0, map[rune]byte{}
		for _, c := range e.mapping {
			if _, ok := backMapping[c]; !ok {
				backMapping[c] = byte(i)
			}
			for j := range buf {
				buf[j] = 0
			}
			n := utf8.EncodeRune(buf, c)
			if n > 3 {
				panic(fmt.Sprintf("rune %q (%U) is too long", c, c))
			}
			fmt.Printf("{%d,[3]byte{0x%02x,0x%02x,0x%02x}},", n, buf[0], buf[1], buf[2])
			if i%2 == 1 {
				fmt.Printf("\n")
			}
			i++
		}
		fmt.Printf("},\n")

		fmt.Printf("encode: [256]uint32{\n")
		encode := make([]uint32, 0, 256)
		for c, i := range backMapping {
			encode = append(encode, uint32(i)<<24|uint32(c))
		}
		sort.Sort(byRune(encode))
		for len(encode) < cap(encode) {
			encode = append(encode, encode[len(encode)-1])
		}
		for i, enc := range encode {
			fmt.Printf("0x%08x,", enc)
			if i%8 == 7 {
				fmt.Printf("\n")
			}
		}
		fmt.Printf("},\n}\n")
	}
	// TODO: add proper line breaking.
	fmt.Printf("var listAll = []encoding.Encoding{\n%s,\n}\n\n", strings.Join(all, ",\n"))
}

type byRune []uint32

func (b byRune) Len() int           { return len(b) }
func (b byRune) Less(i, j int) bool { return b[i]&0xffffff < b[j]&0xffffff }
func (b byRune) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
