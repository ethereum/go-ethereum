// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"testing"
)

// parseMessageRef is the encoding/json based implementation that parseMessage
// replaced. The tests below require the two to agree.
func parseMessageRef(raw json.RawMessage) ([]*jsonrpcMessage, bool) {
	if !isBatch(raw) {
		msgs := []*jsonrpcMessage{{}}
		json.Unmarshal(raw, &msgs[0])
		return msgs, false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.Token() // skip '['
	var msgs []*jsonrpcMessage
	for dec.More() {
		msgs = append(msgs, new(jsonrpcMessage))
		dec.Decode(&msgs[len(msgs)-1])
	}
	return msgs, true
}

// parsePositionalArgumentsRef is the implementation parsePositionalArguments
// replaced.
func parsePositionalArgumentsRef(rawArgs json.RawMessage, types []reflect.Type) ([]reflect.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(rawArgs))
	var args []reflect.Value
	tok, err := dec.Token()
	switch {
	case err == io.EOF || tok == nil && err == nil:
	case err != nil:
		return nil, err
	case tok == json.Delim('['):
		args = make([]reflect.Value, 0, len(types))
		for i := 0; dec.More(); i++ {
			if i >= len(types) {
				return args, fmt.Errorf("too many arguments, want at most %d", len(types))
			}
			argval := reflect.New(types[i])
			if err := dec.Decode(argval.Interface()); err != nil {
				return args, fmt.Errorf("invalid argument %d: %v", i, err)
			}
			if argval.IsNil() && types[i].Kind() != reflect.Pointer {
				return args, fmt.Errorf("missing value for required argument %d", i)
			}
			args = append(args, argval.Elem())
		}
		if _, err := dec.Token(); err != nil {
			return args, err
		}
	default:
		return nil, errUnknownArgs
	}
	for i := len(args); i < len(types); i++ {
		if types[i].Kind() != reflect.Pointer {
			return nil, fmt.Errorf("missing value for required argument %d", i)
		}
		args = append(args, reflect.Zero(types[i]))
	}
	return args, nil
}

var errUnknownArgs = fmt.Errorf("non-array args")

// messageCorpus holds inputs that the two implementations must agree on. Every
// entry is valid JSON, which is what parseMessage is given.
var messageCorpus = []string{
	`{}`,
	`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`,
	`{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":"0x00"},"latest"]}`,
	`{"id":null}`,
	`{"result":null}`,
	`{"params":null}`,
	`{"jsonrpc":"2.0","id":1,"result":null}`,
	`{"id":"str-id","method":"m"}`,
	`{"id":1.5e3,"method":"m"}`,
	`{"id":true,"method":"m"}`,
	// escapes in the string fields
	`{"method":"a\"b","id":1}`,
	`{"method":"a\\b","id":1}`,
	`{"method":"hello","id":1}`,
	`{"method":"tab\there","id":1}`,
	`{"method":"back\\\\slash","id":1}`,
	// structural bytes inside strings must not confuse the scan
	`{"method":"m","params":["{[,:}]"],"id":1}`,
	`{"method":"m","params":["a\"},{\"b"],"id":1}`,
	`{"method":"m","params":["ends with backslash\\"],"id":1}`,
	// whitespace
	"{ \"method\" : \"m\" , \"id\" : 1 }",
	"\n\t{\"method\":\"m\",\"id\":2}\r\n",
	`{"params":  [  1  ,  2  ]  ,"id":3}`,
	// duplicate keys, last one wins
	`{"method":"first","method":"second","id":1}`,
	`{"id":1,"id":2}`,
	// unknown fields are ignored
	`{"method":"m","id":1,"extra":{"a":[1,2,3]},"more":"x"}`,
	// nested params
	`{"method":"m","id":1,"params":[[[[1]]]],"x":1}`,
	`{"method":"m","id":1,"params":[{"a":{"b":{"c":[]}}}]}`,
	// empty and odd values
	`{"method":"","id":1}`,
	`{"":1,"method":"m"}`,
	// not an object at all
	`1`,
	`"str"`,
	`null`,
	`true`,
	// batches
	`[]`,
	`[{"method":"a","id":1}]`,
	`[{"method":"a","id":1},{"method":"b","id":2}]`,
	`[null]`,
	`[{"method":"a","id":1},null,{"method":"b","id":2}]`,
	`[1,2,3]`,
	`["a","b"]`,
	`[[1],[2]]`,
	`[ { "method" : "a" , "id" : 1 } , null ]`,
	`[{"method":"m","params":["},{"]}]`,
}

func TestParseMessageMatchesReference(t *testing.T) {
	for _, input := range messageCorpus {
		if !json.Valid([]byte(input)) {
			t.Fatalf("corpus entry is not valid JSON: %s", input)
		}
		wantMsgs, wantBatch := parseMessageRef(json.RawMessage(input))
		gotMsgs, gotBatch := parseMessage(json.RawMessage(input))
		if gotBatch != wantBatch {
			t.Errorf("%s: batch = %v, want %v", input, gotBatch, wantBatch)
			continue
		}
		if len(gotMsgs) != len(wantMsgs) {
			t.Errorf("%s: got %d messages, want %d", input, len(gotMsgs), len(wantMsgs))
			continue
		}
		for i := range wantMsgs {
			if (gotMsgs[i] == nil) != (wantMsgs[i] == nil) {
				t.Errorf("%s: message %d nil = %v, want %v", input, i, gotMsgs[i] == nil, wantMsgs[i] == nil)
				continue
			}
			if gotMsgs[i] == nil {
				continue
			}
			if err := sameMessage(gotMsgs[i], wantMsgs[i]); err != nil {
				t.Errorf("%s: message %d: %v", input, i, err)
			}
		}
	}
}

func sameMessage(got, want *jsonrpcMessage) error {
	if got.Version != want.Version {
		return fmt.Errorf("Version = %q, want %q", got.Version, want.Version)
	}
	if got.Method != want.Method {
		return fmt.Errorf("Method = %q, want %q", got.Method, want.Method)
	}
	for _, f := range []struct {
		name      string
		got, want json.RawMessage
	}{
		{"ID", got.ID, want.ID},
		{"Params", got.Params, want.Params},
		{"Error", got.Error, want.Error},
		{"Result", got.Result, want.Result},
	} {
		if (f.got == nil) != (f.want == nil) {
			return fmt.Errorf("%s nil = %v, want %v (got %q want %q)", f.name, f.got == nil, f.want == nil, f.got, f.want)
		}
		// The raw fields are byte identical apart from surrounding whitespace,
		// which the reference trims and the scan does not carry either.
		if !bytes.Equal(bytes.TrimSpace(f.got), bytes.TrimSpace(f.want)) {
			return fmt.Errorf("%s = %q, want %q", f.name, f.got, f.want)
		}
	}
	return nil
}

// argCorpus holds argument arrays paired with the argument types to decode them
// into.
var argCorpus = []struct {
	args  string
	types []reflect.Type
}{
	{`[]`, nil},
	{`[1]`, []reflect.Type{reflect.TypeOf(int(0))}},
	{`[1,2]`, []reflect.Type{reflect.TypeOf(int(0)), reflect.TypeOf(int(0))}},
	{`[1,2,3]`, []reflect.Type{reflect.TypeOf(int(0))}},
	{`["a"]`, []reflect.Type{reflect.TypeOf("")}},
	{`[null]`, []reflect.Type{reflect.TypeOf(new(int))}},
	{`[null]`, []reflect.Type{reflect.TypeOf(int(0))}},
	{`[{"a":1}]`, []reflect.Type{reflect.TypeOf(map[string]int{})}},
	{`[[1,2],[3]]`, []reflect.Type{reflect.TypeOf([]int{}), reflect.TypeOf([]int{})}},
	{`[ 1 , 2 ]`, []reflect.Type{reflect.TypeOf(int(0)), reflect.TypeOf(int(0))}},
	{`[1]`, []reflect.Type{reflect.TypeOf(int(0)), reflect.TypeOf(new(int))}},
	{`[1]`, []reflect.Type{reflect.TypeOf(int(0)), reflect.TypeOf(int(0))}},
	{`["not an int"]`, []reflect.Type{reflect.TypeOf(int(0))}},
	{`[]`, []reflect.Type{reflect.TypeOf(int(0))}},
	{`["},{"]`, []reflect.Type{reflect.TypeOf("")}},
	// a type that decodes itself, which is the case the fast path changes
	{`["0x1234"]`, []reflect.Type{reflect.TypeOf(selfDecoding{})}},
	{`[null]`, []reflect.Type{reflect.TypeOf(selfDecoding{})}},
	{`["bad"]`, []reflect.Type{reflect.TypeOf(selfDecoding{})}},
}

// selfDecoding stands in for the argument types that unmarshal themselves, such
// as an engine API payload or a hexutil value.
type selfDecoding struct {
	Text string
}

func (s *selfDecoding) UnmarshalJSON(input []byte) error {
	if len(input) < 2 || input[0] != '"' {
		return fmt.Errorf("selfDecoding: not a string: %s", input)
	}
	s.Text = string(input[1 : len(input)-1])
	return nil
}

func TestParsePositionalArgumentsMatchesReference(t *testing.T) {
	for _, tc := range argCorpus {
		if !json.Valid([]byte(tc.args)) {
			t.Fatalf("corpus entry is not valid JSON: %s", tc.args)
		}
		wantArgs, wantErr := parsePositionalArgumentsRef(json.RawMessage(tc.args), tc.types)
		gotArgs, gotErr := parsePositionalArguments(json.RawMessage(tc.args), tc.types)

		if (gotErr == nil) != (wantErr == nil) {
			t.Errorf("%s: err = %v, want %v", tc.args, gotErr, wantErr)
			continue
		}
		if gotErr != nil {
			continue
		}
		if len(gotArgs) != len(wantArgs) {
			t.Errorf("%s: got %d args, want %d", tc.args, len(gotArgs), len(wantArgs))
			continue
		}
		for i := range wantArgs {
			g, w := gotArgs[i].Interface(), wantArgs[i].Interface()
			if !reflect.DeepEqual(g, w) {
				t.Errorf("%s: arg %d = %#v, want %#v", tc.args, i, g, w)
			}
		}
	}
}

// TestParsePositionalArgumentsEmpty covers the inputs that reach the function
// when a request carries no params at all.
func TestParsePositionalArgumentsEmpty(t *testing.T) {
	for _, args := range []string{"", " ", "null", "  null  "} {
		got, err := parsePositionalArguments(json.RawMessage(args), nil)
		if err != nil {
			t.Errorf("%q: unexpected error %v", args, err)
		}
		if len(got) != 0 {
			t.Errorf("%q: got %d args, want 0", args, len(got))
		}
	}
}

// benchMessages are requests of the shapes the server actually sees, from a
// one line call to a payload sized one.
func benchMessages() []struct {
	name string
	req  string
} {
	bigArg := func(n int) string {
		var buf bytes.Buffer
		buf.WriteString(`{"parentHash":"0x1234","transactions":[`)
		for i := 0; i < n; i++ {
			if i > 0 {
				buf.WriteByte(',')
			}
			buf.WriteByte('"')
			buf.WriteString("0x")
			for j := 0; j < 1024; j++ {
				buf.WriteString("ab")
			}
			buf.WriteByte('"')
		}
		buf.WriteString(`]}`)
		return buf.String()
	}
	return []struct {
		name string
		req  string
	}{
		{"tiny", `{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}`},
		{"small", `{"jsonrpc":"2.0","id":1,"method":"eth_getBlockByNumber","params":["0x1b4",true]}`},
		{"batch10", func() string {
			var buf bytes.Buffer
			buf.WriteByte('[')
			for i := 0; i < 10; i++ {
				if i > 0 {
					buf.WriteByte(',')
				}
				fmt.Fprintf(&buf, `{"jsonrpc":"2.0","id":%d,"method":"eth_chainId","params":[]}`, i)
			}
			buf.WriteByte(']')
			return buf.String()
		}()},
		{"payload64kb", fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"engine_newPayloadV4","params":[%s,[],null,[]]}`, bigArg(32))},
		{"payload512kb", fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"engine_newPayloadV4","params":[%s,[],null,[]]}`, bigArg(256))},
	}
}

// BenchmarkParseMessage compares the scan based envelope split against the
// encoding/json one it replaced.
func BenchmarkParseMessage(b *testing.B) {
	for _, tc := range benchMessages() {
		raw := json.RawMessage(tc.req)
		b.Run(tc.name+"/new", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				parseMessage(raw)
			}
		})
		b.Run(tc.name+"/ref", func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				parseMessageRef(raw)
			}
		})
	}
}

// BenchmarkParsePositionalArguments compares the two argument decoders on an
// argument that decodes itself, which is the shape an engine API payload has.
func BenchmarkParsePositionalArguments(b *testing.B) {
	types := []reflect.Type{reflect.TypeOf(selfDecoding{})}
	for _, size := range []int{1, 64, 512} {
		var buf bytes.Buffer
		buf.WriteString(`["0x`)
		for i := 0; i < size*512; i++ {
			buf.WriteString("ab")
		}
		buf.WriteString(`"]`)
		raw := json.RawMessage(buf.String())
		b.Run(fmt.Sprintf("kb%d/new", len(raw)/1024), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				if _, err := parsePositionalArguments(raw, types); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(fmt.Sprintf("kb%d/ref", len(raw)/1024), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				if _, err := parsePositionalArgumentsRef(raw, types); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// FuzzJSONScanFields checks that the field scan agrees with encoding/json on any
// valid JSON object.
func FuzzJSONScanFields(f *testing.F) {
	for _, s := range messageCorpus {
		f.Add(s)
	}
	f.Add(`{"a":"😀","b":[1,{"c":null}]}`)
	f.Fuzz(func(t *testing.T, input string) {
		data := []byte(input)
		if !json.Valid(data) {
			return
		}
		var want map[string]json.RawMessage
		if err := json.Unmarshal(data, &want); err != nil {
			return // not an object
		}
		got := make(map[string]json.RawMessage)
		forEachJSONField(data, func(key, value []byte) {
			// The scan hands back the key still escaped, so unescape it the same
			// way the map decode did before comparing.
			var k string
			if err := json.Unmarshal(append(append([]byte{'"'}, key...), '"'), &k); err != nil {
				t.Fatalf("key %q does not unescape: %v", key, err)
			}
			got[k] = value
		})
		if len(got) != len(want) {
			t.Fatalf("got %d fields, want %d (input %s)", len(got), len(want), input)
		}
		for k, wv := range want {
			gv, ok := got[k]
			if !ok {
				t.Fatalf("missing field %q (input %s)", k, input)
			}
			if !bytes.Equal(bytes.TrimSpace(gv), bytes.TrimSpace(wv)) {
				t.Fatalf("field %q = %q, want %q (input %s)", k, gv, wv, input)
			}
		}
	})
}

// FuzzJSONScanElements checks that the element scan agrees with encoding/json on
// any valid JSON array.
func FuzzJSONScanElements(f *testing.F) {
	for _, s := range messageCorpus {
		f.Add(s)
	}
	f.Add(`[1,"two",{"three":3},[4],null,true]`)
	f.Fuzz(func(t *testing.T, input string) {
		data := []byte(input)
		if !json.Valid(data) {
			return
		}
		var want []json.RawMessage
		if err := json.Unmarshal(data, &want); err != nil {
			return // not an array
		}
		var got []json.RawMessage
		forEachJSONElement(data, func(value []byte) {
			got = append(got, value)
		})
		if len(got) != len(want) {
			t.Fatalf("got %d elements, want %d (input %s)", len(got), len(want), input)
		}
		for i := range want {
			if !bytes.Equal(bytes.TrimSpace(got[i]), bytes.TrimSpace(want[i])) {
				t.Fatalf("element %d = %q, want %q (input %s)", i, got[i], want[i], input)
			}
		}
	})
}
