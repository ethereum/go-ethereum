# common

[![Build
Status](https://travis-ci.org/ethereum/go-ethereum.png?branch=master)](https://travis-ci.org/ethereum/go-ethereum)

The common package contains the ethereum utility library.

# Installation

As a subdirectory the main go-ethereum repository, you get it with
`go get github.com/ethereum/go-ethereum`.

# Usage

## RLP (Recursive Linear Prefix) Encoding

RLP Encoding is an encoding scheme used by the Ethereum project. It
encodes any native value or list to a string.

More in depth information about the encoding scheme see the
[Wiki](http://wiki.ethereum.org/index.php/RLP) article.

```go
rlp := common.Encode("doge")
fmt.Printf("%q\n", rlp) // => "\0x83dog"

rlp = common.Encode([]interface{}{"dog", "cat"})
fmt.Printf("%q\n", rlp) // => "\0xc8\0x83dog\0x83cat"
decoded := common.Decode(rlp)
fmt.Println(decoded) // => ["dog" "cat"]
```

## Patricia Trie

Patricie Tree is a merkle trie used by the Ethereum project.

More in depth information about the (modified) Patricia Trie can be
found on the [Wiki](http://wiki.ethereum.org/index.php/Patricia_Tree).

The patricia trie uses a db as backend and could be anything as long as
it satisfies the Database interface found in `common/db.go`.

```go
db := NewDatabase()

// db, root
trie := common.NewTrie(db, "")

trie.Put("puppy", "dog")
trie.Put("horse", "stallion")
trie.Put("do", "verb")
trie.Put("doge", "coin")

// Look up the key "do" in the trie
out := trie.Get("do")
fmt.Println(out) // => verb

trie.Delete("puppy")
```

The patricia trie, in combination with RLP, provides a robust,
cryptographically authenticated data structure that can be used to store
all (key, value) bindings.

```go
// ... Create db/trie

// Note that RLP uses interface slices as list
value := common.Encode([]interface{}{"one", 2, "three", []interface{}{42}})
// Store the RLP encoded value of the list
trie.Put("mykey", value)
```

## Value

Value is a Generic Value which is used in combination with RLP data or
`([])interface{}` structures. It may serve as a bridge between RLP data
and actual real values and takes care of all the type checking and
casting. Unlike Go's `reflect.Value` it does not panic if it's unable to
cast to the requested value. It simple returns the base value of that
type (e.g. `Slice()` returns []interface{}, `Uint()` return 0, etc).

### Creating a new Value

`NewEmptyValue()` returns a new \*Value with it's initial value set to a
`[]interface{}`

`AppendList()` appends a list to the current value.

`Append(v)` appends the value (v) to the current value/list.

```go
val := common.NewEmptyValue().Append(1).Append("2")
val.AppendList().Append(3)
```

### Retrieving values

`Get(i)` returns the `i` item in the list.

`Uint()` returns the value as an unsigned int64.

`Slice()` returns the value as a interface slice.

`Str()` returns the value as a string.

`Bytes()` returns the value as a byte slice.

`Len()` assumes current to be a slice and returns its length.

`Byte()` returns the value as a single byte.

```go
val := common.NewValue([]interface{}{1,"2",[]interface{}{3}})
val.Get(0).Uint() // => 1
val.Get(1).Str()  // => "2"
s := val.Get(2)   // => Value([]interface{}{3})
s.Get(0).Uint()   // => 3
```

## Decoding

Decoding streams of RLP data is simplified

```go
val := common.NewValueFromBytes(rlpData)
val.Get(0).Uint()
```

## Encoding

Encoding from Value to RLP is done with the `Encode` method. The
underlying value can be anything RLP can encode (int, str, lists, bytes)

```go
val := common.NewValue([]interface{}{1,"2",[]interface{}{3}})
rlp := val.Encode()
// Store the rlp data
Store(rlp)
```
