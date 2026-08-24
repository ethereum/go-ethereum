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
	"reflect"
	"strings"
	"testing"
)

// messageCorpus seeds the fuzz targets below. Every entry is valid JSON, which
// is the state of a message by the time parseMessage sees it.
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
	// keys in other spellings are unknown fields, only the exact names match
	`{"METHOD":"m","ID":1}`,
	`{"Method":"m","Id":1,"Jsonrpc":"2.0"}`,
	"{\"metho\\u0064\":\"m\",\"i\\u0064\":7}",
	`{"metho\\u0064":"not the method key"}`,
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

// TestParseMessage covers the inputs where the envelope split has to make a
// decision.
func TestParseMessage(t *testing.T) {
	msg := func(version, method, id, params, result string) *jsonrpcMessage {
		m := &jsonrpcMessage{Version: version, Method: method}
		if id != "" {
			m.ID = json.RawMessage(id)
		}
		if params != "" {
			m.Params = json.RawMessage(params)
		}
		if result != "" {
			m.Result = json.RawMessage(result)
		}
		return m
	}
	zero := func() *jsonrpcMessage { return msg("", "", "", "", "") }

	tests := []struct {
		name  string
		input string
		batch bool
		want  []*jsonrpcMessage
	}{
		{"empty object", `{}`, false, []*jsonrpcMessage{zero()}},
		{
			"call",
			`{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}`,
			false, []*jsonrpcMessage{msg("2.0", "eth_chainId", "1", "[]", "")},
		},
		// null stays in the raw fields as the literal. isResponse needs a null
		// result to count as present.
		{"null id", `{"id":null}`, false, []*jsonrpcMessage{msg("", "", "null", "", "")}},
		{"null result", `{"result":null}`, false, []*jsonrpcMessage{msg("", "", "", "", "null")}},
		{"null params", `{"params":null}`, false, []*jsonrpcMessage{msg("", "", "", "null", "")}},

		{"escaped quote in method", `{"method":"a\"b","id":1}`, false, []*jsonrpcMessage{msg("", `a"b`, "1", "", "")}},
		{"escaped tab in method", `{"method":"tab\there","id":1}`, false, []*jsonrpcMessage{msg("", "tab\there", "1", "", "")}},
		{"duplicate key, last wins", `{"method":"first","method":"second","id":1}`, false, []*jsonrpcMessage{msg("", "second", "1", "", "")}},

		// field names have one spelling in the spec, any other spelling is an
		// unknown key, even where encoding/json would have matched it
		{"cased keys ignored", `{"Method":"m","ID":1,"Params":[1]}`, false, []*jsonrpcMessage{zero()}},
		{"upper case keys ignored", `{"METHOD":"m","JSONRPC":"2.0"}`, false, []*jsonrpcMessage{zero()}},
		{"escaped keys ignored", "{\"metho\\u0064\":\"m\",\"i\\u0064\":7}", false, []*jsonrpcMessage{zero()}},

		// a string holding structural bytes must not end the value early
		{
			"structural bytes inside a string",
			`{"method":"m","params":["a\"},{\"b"],"id":1}`,
			false, []*jsonrpcMessage{msg("", "m", "1", `["a\"},{\"b"]`, "")},
		},
		{"space inside params is kept", `{"params":  [  1  ,  2  ]  ,"id":3}`, false, []*jsonrpcMessage{msg("", "", "3", "[  1  ,  2  ]", "")}},
		{"space around the message", "\n\t{\"method\":\"m\",\"id\":2}\r\n", false, []*jsonrpcMessage{msg("", "m", "2", "", "")}},
		{"unknown fields ignored", `{"method":"m","id":1,"extra":{"a":[1,2,3]},"more":"x"}`, false, []*jsonrpcMessage{msg("", "m", "1", "", "")}},

		// valid JSON that is not a message leaves a zero one for the handler to reject
		{"not an object", `1`, false, []*jsonrpcMessage{zero()}},
		{"bare null", `null`, false, []*jsonrpcMessage{nil}},

		{"empty batch", `[]`, true, nil},
		{"batch holding one null", `[null]`, true, []*jsonrpcMessage{nil}},
		{
			"batch with a null in it",
			`[{"method":"a","id":1},null,{"method":"b","id":2}]`,
			true, []*jsonrpcMessage{msg("", "a", "1", "", ""), nil, msg("", "b", "2", "", "")},
		},
		{"batch of non-objects", `[1,2,3]`, true, []*jsonrpcMessage{zero(), zero(), zero()}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !json.Valid([]byte(tc.input)) {
				t.Fatalf("test input is not valid JSON: %s", tc.input)
			}
			got, batch := parseMessage(json.RawMessage(tc.input))
			if batch != tc.batch {
				t.Fatalf("batch = %v, want %v", batch, tc.batch)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d messages, want %d", len(got), len(tc.want))
			}
			for i := range tc.want {
				if (got[i] == nil) != (tc.want[i] == nil) {
					t.Fatalf("message %d nil = %v, want %v", i, got[i] == nil, tc.want[i] == nil)
				}
				if got[i] == nil {
					continue
				}
				if err := sameMessage(got[i], tc.want[i]); err != nil {
					t.Errorf("message %d: %v", i, err)
				}
			}
		})
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
		// A raw field is a slice of the input, so it should come back byte for byte.
		if !bytes.Equal(f.got, f.want) {
			return fmt.Errorf("%s = %q, want %q", f.name, f.got, f.want)
		}
	}
	return nil
}

// selfDecoding stands in for an argument type that unmarshals itself.
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

// TestParsePositionalArguments covers how an argument array is cut up, including
// the cases that error.
func TestParsePositionalArguments(t *testing.T) {
	var (
		tInt  = reflect.TypeOf(int(0))
		tPtr  = reflect.TypeOf(new(int))
		tStr  = reflect.TypeOf("")
		tMap  = reflect.TypeOf(map[string]int{})
		tSelf = reflect.TypeOf(selfDecoding{})
	)
	tests := []struct {
		name    string
		args    string
		types   []reflect.Type
		want    []any
		wantErr string
	}{
		{"no arguments", `[]`, nil, nil, ""},
		{"two ints", `[1,2]`, []reflect.Type{tInt, tInt}, []any{1, 2}, ""},
		{"space between arguments", `[ 1 , 2 ]`, []reflect.Type{tInt, tInt}, []any{1, 2}, ""},
		{"object argument", `[{"a":1}]`, []reflect.Type{tMap}, []any{map[string]int{"a": 1}}, ""},
		{"structural bytes inside a string", `["},{"]`, []reflect.Type{tStr}, []any{`},{`}, ""},
		{"null into a pointer", `[null]`, []reflect.Type{tPtr}, []any{(*int)(nil)}, ""},
		{"null into a value", `[null]`, []reflect.Type{tInt}, []any{0}, ""},
		{"missing optional argument", `[1]`, []reflect.Type{tInt, tPtr}, []any{1, (*int)(nil)}, ""},

		{"too many arguments", `[1,2,3]`, []reflect.Type{tInt}, nil, "too many arguments"},
		{"missing required argument", `[1]`, []reflect.Type{tInt, tInt}, nil, "missing value for required argument 1"},
		{"no arguments at all", `[]`, []reflect.Type{tInt}, nil, "missing value for required argument 0"},
		{"wrong type", `["not an int"]`, []reflect.Type{tInt}, nil, "invalid argument 0"},

		// a self decoding type is handed the value directly, except for null
		{"self decoding", `["0x1234"]`, []reflect.Type{tSelf}, []any{selfDecoding{Text: "0x1234"}}, ""},
		{"self decoding null", `[null]`, []reflect.Type{tSelf}, nil, "invalid argument 0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !json.Valid([]byte(tc.args)) {
				t.Fatalf("test input is not valid JSON: %s", tc.args)
			}
			got, err := parsePositionalArguments(json.RawMessage(tc.args), tc.types)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d arguments, want %d", len(got), len(tc.want))
			}
			for i := range tc.want {
				if v := got[i].Interface(); !reflect.DeepEqual(v, tc.want[i]) {
					t.Errorf("argument %d = %#v, want %#v", i, v, tc.want[i])
				}
			}
		})
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

// benchMessages are request shapes the server sees, from a one line call to a
// payload sized one.
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

// BenchmarkParseMessage measures the envelope split.
func BenchmarkParseMessage(b *testing.B) {
	for _, tc := range benchMessages() {
		raw := json.RawMessage(tc.req)
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				parseMessage(raw)
			}
		})
	}
}

// BenchmarkParsePositionalArguments measures argument decoding for a type that
// decodes itself, which is the shape an engine API payload has.
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
		b.Run(fmt.Sprintf("kb%d", len(raw)/1024), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				if _, err := parsePositionalArguments(raw, types); err != nil {
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

// FuzzFillMessage checks the envelope split against encoding/json field by
// field. Field names have one spelling, so the reference picks each one out of
// a decoded map by its exact name.
func FuzzFillMessage(f *testing.F) {
	for _, s := range messageCorpus {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		data := []byte(input)
		if !json.Valid(data) {
			return
		}
		if bytes.IndexByte(data, '\\') >= 0 {
			// encoding/json unescapes map keys, so an escaped key would match
			// in the reference but is an unknown key to fillMessage. Escape
			// handling is pinned by the tests above.
			return
		}
		var want jsonrpcMessage
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err == nil {
			if v, ok := obj["jsonrpc"]; ok {
				json.Unmarshal(v, &want.Version)
			}
			if v, ok := obj["id"]; ok {
				want.ID = v
			}
			if v, ok := obj["method"]; ok {
				json.Unmarshal(v, &want.Method)
			}
			if v, ok := obj["params"]; ok {
				want.Params = v
			}
			if v, ok := obj["error"]; ok {
				want.Error = v
			}
			if v, ok := obj["result"]; ok {
				want.Result = v
			}
		}
		got := new(jsonrpcMessage)
		fillMessage(data, got)
		if err := sameMessage(got, &want); err != nil {
			t.Fatalf("%v (input %s)", err, input)
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
