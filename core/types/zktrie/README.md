# Type for zktrie 

## Data Format in stateDb

All data node being stored via stateDb are encoded by following syntax:

``` EBNF
    node             =  magic string | node data ;

    magic string     =  "THIS IS SOME MAGIC BYTES FOR SMT m1rRXgP2xpDI" ;

    node data        =  middle node | leaf node | empty node ;

    empty node       = '0x2' ;

    middle node      = '0x0', left hash, right hash ;

    field            = 32 * hex char ;

    left hash        = field ;

    right hash       = field ;

    leaf node        = node key , value len , compress flag , <value len> * value field, key preimage ;

    node key         = field ;

    compress flag    = 3 * byte ;

    value len        = byte ;

    value field      = field | compressed field, compressed field ;

    compressed field = 16 * hex char ;

    key preimage     = '0x0' | preimage bytes ;

    preimage bytes   = len, <len> * byte ;

    len              = byte ;
```

A `field` is an element in prime field of BN256 represented by **big endian** integer and contained in fixed length (32) bytes; 

A `compressed field` is a field represented by **big endian** integer which could be contained in 16 bytes;

For the total `value len` items of `value field` (maximum 255), the first 24 `value field`s can be recorded as `field` or 2x `compressed field` (i.e. a byte32). The corresonpdoing bit in `compress flag` is set to 1 if it was recorded as byte32, or 0 for a field.

## Key scheme

The key of data node is obtained from one or more poseidon hash calculation: `poseidon := (field, field) => field`.

For middle node:

```
key = poseidon(<left hash>, <right hash>)
```

For leaf node:

```
key = poseidon(<pre key>, <value hash>)

pre key = poseidon(field(1), <node key>)

value hash = poseidon(<leaf element>, <leaf element>) | poseidon(<value hash>, <value hash>)

leaf element = <value field as field> | poseidon(<compressed field as field>, <compressed field as field>) | field(0)

```

That is, to calculate the key of a leaf node:

1. In the sequence of `value field`s, take which is recorded as 'compressed' and calculate the 2x `compressed field` for its poseidon hash, replace the corresponding `value field` ad-hoc in the sequence;

2. Consider the sequence from 1 as the leafs of a binary merkle tree (append a 0 field for odd leafs) and calculate its root by poseidon hash;

For empty node:

```
key = field(0)
```

## Account data

Each account data is saved in one leaf node of account zktrie as 4 `value field`s:

1. Nonce as `field`
2. Balance as `field`
3. CodeHash as `compressed field` (byte32)
4. Storage root as `field`

The key for an account data is calculated from the 20-bit account address as following:

```

32-byte-zero-end-padding-addr := address, 16 * bytes (0)

key = poseidon(<first 16 byte of 32-byte-zero-end-padding-addr as field>, <last 16 byte of 32-byte-zero-end-padding-addr as field>)

```

## Data examples

### A leaf node in account trie:

> 0x017f9d3bbc51d12566ecc6049ca6bf76e32828c22b197405f63a833b566fe7da0a040400000000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000029b74e075daad9f17eb39cd893c2dd32f52ecd99084d63964842defd00ebcbe208a2f471d50e56ac5000ab9e82f871e36b5a636b19bd02f70aa666a3bd03142f00

Can be decompose to:

+ `0x01`: node type prefix for leaf node
+ `7f9d3bbc51d12566ecc6049ca6bf76e32828c22b197405f63a833b566fe7da0a`: node key as field
+ `04`: value len (4 value fields)
+ `040000`: compress flag, a 24 bit array, indicating the third field is compressed
+ `0000000000000000000000000000000000000000000000000000000000000001`: value field 0 (nonce)
+ `0000000000000000000000000000000000000000000000000000000000000000`: value field 1 (balance)
+ `29b74e075daad9f17eb39cd893c2dd32f52ecd99084d63964842defd00ebcbe2`: value field 2 (codeHash, as byte32)
+ `08a2f471d50e56ac5000ab9e82f871e36b5a636b19bd02f70aa666a3bd03142f`: value field 3 (storage root)
+ `00`: key preimage is not avaliable

The key calculation for this node is:

```

arr = [<value field 0>, <value field 1>, <value field 2>, <value field 3>]

hash_pre = poseidon(<first 16 byte for value field 2>, <last 16 byte for value field 2>)

arr[2] = hash_pre

layer1 = [poseidon(arr[0], arr[1]), poseidon(arr[2], arr[3])]

key = poseidon(layer1[0], layer1[1])

```

Notice all field and compressed field are represented as **big endian** integer.

### A middle node in account trie:

> 0x00000000000000000000000000000000000000000000000000000000000000000004470b58d80eeb26da85b2c2db5c254900656fb459c07729f556ff02534ab32a

Notice the left child of this node is an empty node (so its key is field(0))