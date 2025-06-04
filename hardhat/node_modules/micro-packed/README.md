# micro-packed

> Less painful binary encoding / decoding

Define complex binary structures using composable primitives. Comes with a friendly [debugger](#debugger).

Used in:

- [btc-signer](https://github.com/paulmillr/scure-btc-signer) for parsing of Bitcoin Script
- [eth-signer](https://github.com/paulmillr/micro-eth-signer) for RLP and SSZ decoding. RLP pointers are protected against DoS
- [sol-signer](https://github.com/paulmillr/micro-sol-signer) for parsing of keys, messages and other things
- [micro-ordinals](https://github.com/paulmillr/micro-ordinals) for Bitcoin ordinal parsing
- [key-producer](https://github.com/paulmillr/micro-key-producer) for lightweight implementations of PGP, SSH and OTP

## Usage

> `npm install micro-packed`

> `jsr add jsr:@paulmillr/micro-packed`

```ts
import * as P from 'micro-packed';
const s = P.struct({
  field1: P.U32BE, // 32-bit unsigned big-endian integer
  field2: P.string(P.U8), // String with U8 length prefix
  field3: P.bytes(32), // 32 bytes
  field4: P.array(
    P.U16BE,
    P.struct({
      // Array of structs with U16BE length
      subField1: P.U64BE, // 64-bit unsigned big-endian integer
      subField2: P.string(10), // 10-byte string
    })
  ),
});
```

Table of contents:

- [Basics](#basics)
- Primitive types: [P.bytes](#pbytes), [P.string](#pstring), [P.hex](#phex), [P.constant](#pconstant), [P.pointer](#ppointer)
- Complex types: [P.array](#parray), [P.struct](#pstruct), [P.tuple](#ptuple), [P.map](#pmap), [P.tag](#ptag), [P.mappedTag](#pmappedtag)
- Padding, prefix, magic: [P.padLeft](#ppadleft), [P.padRight](#ppadright), [P.prefix](#pprefix), [P.magic](#pmagic), [P.magicBytes](#pmagicbytes)
- Flags: [P.flag](#pflag), [P.flagged](#pflagged), [P.optional](#poptional)
- Wrappers: [P.apply](#papply), [P.wrap](#pwrap), [P.lazy](#plazy)
- Bit fiddling: [P.bits](#pbits), [P.bitset](#pbitset)
- [utils](#utils): [P.validate](#pvalidate), [coders.decimal](#codersdecimal)
- [Debugger](#debugger)

### Basics

There are 3 main interfaces:

- `Coder<F, T>` - a converter between types F and T
- `BytesCoder<T>` - a Coder from type T to Bytes
- `BytesCoderStream<T>` - streaming BytesCoder with Reader and Writer streams

Coder and BytesCoder use `encode` / `decode` methods

BytesCoderStream use `encodeStream` and `decodeStream`

#### Flexible size

Many primitives accept length / size / len as their argument.
It represents their size. There are four different types of size:

- CoderType: Dynamic size (prefixed with a length CoderType like U16BE)
- number: Fixed size (specified by a number)
- terminator: Uint8Array (will parse until these bytes are matched)
- null: (null, will parse until end of buffer)

## Primitive types

### P.bytes

Bytes CoderType with a specified length and endianness.

| Param | Description                                                     |
| ----- | --------------------------------------------------------------- |
| len   | Length CoderType, number, Uint8Array (for terminator), or null. |
| le    | Whether to use little-endian byte order.                        |

```js
// Dynamic size bytes (prefixed with P.U16BE number of bytes length)
const dynamicBytes = P.bytes(P.U16BE, false);
const fixedBytes = P.bytes(32, false); // Fixed size bytes
const unknownBytes = P.bytes(null, false); // Unknown size bytes, will parse until end of buffer
const zeroTerminatedBytes = P.bytes(new Uint8Array([0]), false); // Zero-terminated bytes
```

Following shortcuts are also available:

- `P.EMPTY`: Shortcut to zero-length (empty) byte array
- `P.NULL`: Shortcut to one-element (element is 0) byte array

### P.string

String CoderType with a specified length and endianness.

| Param | Description                                        |
| ----- | -------------------------------------------------- |
| len   | CoderType, number, Uint8Array (terminator) or null |
| le    | Whether to use little-endian byte order.           |

```js
const dynamicString = P.string(P.U16BE, false); // Dynamic size string (prefixed with P.U16BE number of string length)
const fixedString = P.string(10, false); // Fixed size string
const unknownString = P.string(null, false); // Unknown size string, will parse until end of buffer
const nullTerminatedString = P.cstring; // NUL-terminated string
const _cstring = P.string(new Uint8Array([0])); // Same thing
```

### P.hex

Hexadecimal string CoderType with a specified length, endianness, and optional 0x prefix.

**Returns**: CoderType representing the hexadecimal string.

| Param  | Description                                                                                                                 |
| ------ | --------------------------------------------------------------------------------------------------------------------------- |
| len    | Length CoderType (dynamic size), number (fixed size), Uint8Array (for terminator), or null (will parse until end of buffer) |
| isLE   | Whether to use little-endian byte order.                                                                                    |
| with0x | Whether to include the 0x prefix.                                                                                           |

```js
const dynamicHex = P.hex(P.U16BE, { isLE: false, with0x: true }); // Hex string with 0x prefix and U16BE length
const fixedHex = P.hex(32, { isLE: false, with0x: false }); // Fixed-length 32-byte hex string without 0x prefix
```

### P.constant

Creates a CoderType for a constant value. The function enforces this value during encoding,
ensuring it matches the provided constant. During decoding, it always returns the constant value.
The actual value is not written to or read from any byte stream; it's used only for validation.

**Returns**: CoderType representing the constant value.

| Param | Description     |
| ----- | --------------- |
| c     | Constant value. |

```js
// Always return 123 on decode, throws on encoding anything other than 123
const constantU8 = P.constant(123);
```

### P.pointer

Pointer to a value using a pointer CoderType and an inner CoderType.
Pointers are scoped, and the next pointer in the dereference chain is offset by the previous one.
By default (if no 'allowMultipleReads' in ReaderOpts is set) is safe, since
same region of memory cannot be read multiple times.

**Returns**: CoderType representing the pointer to the value.

| Param | Description                                        |
| ----- | -------------------------------------------------- |
| ptr   | CoderType for the pointer value.                   |
| inner | CoderType for encoding/decoding the pointed value. |
| sized | Whether the pointer should have a fixed size.      |

```js
const pointerToU8 = P.pointer(P.U16BE, P.U8); // Pointer to a single U8 value
```

## Complex types

### P.array

Array of items (inner type) with a specified length.

| Param | Description                                                                                                             |
| ----- | ----------------------------------------------------------------------------------------------------------------------- |
| len   | Length CoderType (dynamic size), number (fixed size), Uint8Array (terminator), or null (will parse until end of buffer) |
| inner | CoderType for encoding/decoding each array item.                                                                        |

```js
const a1 = P.array(P.U16BE, child); // Dynamic size array (prefixed with P.U16BE number of array length)
const a2 = P.array(4, child); // Fixed size array
const a3 = P.array(null, child); // Unknown size array, will parse until end of buffer
const a4 = P.array(new Uint8Array([0]), child); // zero-terminated array (NOTE: terminator can be any buffer)
```

### P.struct

Structure of composable primitives (C/Rust struct)

**Returns**: CoderType representing the structure.

| Param  | Description                               |
| ------ | ----------------------------------------- |
| fields | Object mapping field names to CoderTypes. |

```js
// Define a structure with a 32-bit big-endian unsigned integer, a string, and a nested structure
const myStruct = P.struct({
  id: P.U32BE,
  name: P.string(P.U8),
  nested: P.struct({
    flag: P.bool,
    value: P.I16LE,
  }),
});
```

### P.tuple

Tuple (unnamed structure) of CoderTypes. Same as struct but with unnamed fields.

| Param  | Description          |
| ------ | -------------------- |
| fields | Array of CoderTypes. |

```js
const myTuple = P.tuple([P.U8, P.U16LE, P.string(P.U8)]);
```

### P.map

Mapping between encoded values and string representations.

**Returns**: CoderType representing the mapping.

| Param    | Description                                              |
| -------- | -------------------------------------------------------- |
| inner    | CoderType for encoded values.                            |
| variants | Object mapping string representations to encoded values. |

```ts
// Map between numbers and strings
const numberMap = P.map(P.U8, {
  one: 1,
  two: 2,
  three: 3,
});

// Map between byte arrays and strings
const byteMap = P.map(P.bytes(2, false), {
  ab: Uint8Array.from([0x61, 0x62]),
  cd: Uint8Array.from([0x63, 0x64]),
});
```

### P.tag

Tagged union of CoderTypes, where the tag value determines which CoderType to use.
The decoded value will have the structure `{ TAG: number, data: ... }`.

| Param    | Description                              |
| -------- | ---------------------------------------- |
| tag      | CoderType for the tag value.             |
| variants | Object mapping tag values to CoderTypes. |

```js
// Tagged union of array, string, and number
// Depending on the value of the first byte, it will be decoded as an array, string, or number.
const taggedUnion = P.tag(P.U8, {
  0x01: P.array(P.U16LE, P.U8),
  0x02: P.string(P.U8),
  0x03: P.U32BE,
});

const encoded = taggedUnion.encode({ TAG: 0x01, data: 'hello' }); // Encodes the string 'hello' with tag 0x01
const decoded = taggedUnion.decode(encoded); // Decodes the encoded value back to { TAG: 0x01, data: 'hello' }
```

### P.mappedTag

Mapping between encoded values, string representations, and CoderTypes using a tag CoderType.

| Param    | Description                                                            |
| -------- | ---------------------------------------------------------------------- |
| tagCoder | CoderType for the tag value.                                           |
| variants | Object mapping string representations to [tag value, CoderType] pairs. |

```js
const cborValue: P.CoderType<CborValue> = P.mappedTag(P.bits(3), {
  uint: [0, cborUint], // An unsigned integer in the range 0..264-1 inclusive.
  negint: [1, cborNegint], // A negative integer in the range -264..-1 inclusive
  bytes: [2, P.lazy(() => cborLength(P.bytes, cborValue))], // A byte string.
  string: [3, P.lazy(() => cborLength(P.string, cborValue))], // A text string (utf8)
  array: [4, cborArrLength(P.lazy(() => cborValue))], // An array of data items
  map: [5, P.lazy(() => cborArrLength(P.tuple([cborValue, cborValue])))], // A map of pairs of data items
  tag: [6, P.tuple([cborUint, P.lazy(() => cborValue)] as const)], // A tagged data item ("tag") whose tag number
  simple: [7, cborSimple], // Floating-point numbers and simple values, as well as the "break" stop code
});
```

## Padding, prefix, magic

### P.padLeft

Pads a CoderType with a specified block size and padding function on the left side.

**Returns**: CoderType representing the padded value.

| Param     | Description                                                     |
| --------- | --------------------------------------------------------------- |
| blockSize | Block size for padding (positive safe integer).                 |
| inner     | Inner CoderType to pad.                                         |
| padFn     | Padding function to use. If not provided, zero padding is used. |

```js
// Pad a U32BE with a block size of 4 and zero padding
const paddedU32BE = P.padLeft(4, P.U32BE);

// Pad a string with a block size of 16 and custom padding
const paddedString = P.padLeft(16, P.string(P.U8), (i) => i + 1);
```

### P.padRight

Pads a CoderType with a specified block size and padding function on the right side.

**Returns**: CoderType representing the padded value.

| Param     | Description                                                     |
| --------- | --------------------------------------------------------------- |
| blockSize | Block size for padding (positive safe integer).                 |
| inner     | Inner CoderType to pad.                                         |
| padFn     | Padding function to use. If not provided, zero padding is used. |

```js
// Pad a U16BE with a block size of 2 and zero padding
const paddedU16BE = P.padRight(2, P.U16BE);

// Pad a bytes with a block size of 8 and custom padding
const paddedBytes = P.padRight(8, P.bytes(null), (i) => i + 1);
```

### P.ZeroPad

Shortcut to zero-bytes padding

### P.prefix

Prefix-encoded value using a length prefix and an inner CoderType.

**Returns**: CoderType representing the prefix-encoded value.

| Param | Description                                                     |
| ----- | --------------------------------------------------------------- |
| len   | Length CoderType, number, Uint8Array (for terminator), or null. |
| inner | CoderType for the actual value to be prefix-encoded.            |

```js
const dynamicPrefix = P.prefix(P.U16BE, P.bytes(null)); // Dynamic size prefix (prefixed with P.U16BE number of bytes length)
const fixedPrefix = P.prefix(10, P.bytes(null)); // Fixed size prefix (always 10 bytes)
```

### P.magic

Magic value CoderType that encodes/decodes a constant value.
This can be used to check for a specific magic value or sequence of bytes at the beginning of a data structure.

**Returns**: CoderType representing the magic value.

| Param    | Description                                              |
| -------- | -------------------------------------------------------- |
| inner    | Inner CoderType for the value.                           |
| constant | Constant value.                                          |
| check    | Whether to check the decoded value against the constant. |

```js
// Always encodes constant as bytes using inner CoderType, throws if encoded value is not present
const magicU8 = P.magic(P.U8, 0x42);
```

### P.magicBytes

Magic bytes CoderType that encodes/decodes a constant byte array or string.

**Returns**: CoderType representing the magic bytes.

| Param    | Description                    |
| -------- | ------------------------------ |
| constant | Constant byte array or string. |

```js
// Always encodes undefined into byte representation of string 'MAGIC'
const magicBytes = P.magicBytes('MAGIC');
```

## Flags

### P.flag

Flag CoderType that encodes/decodes a boolean value based on the presence of a marker.

**Returns**: CoderType representing the flag value.

| Param     | Description                          |
| --------- | ------------------------------------ |
| flagValue | Marker value.                        |
| xor       | Whether to invert the flag behavior. |

```js
const flag = P.flag(new Uint8Array([0x01, 0x02])); // Encodes true as u8a([0x01, 0x02]), false as u8a([])
const flagXor = P.flag(new Uint8Array([0x01, 0x02]), true); // Encodes true as u8a([]), false as u8a([0x01, 0x02])
// Conditional encoding with flagged
const s = P.struct({ f: P.flag(new Uint8Array([0x0, 0x1])), f2: P.flagged('f', P.U32BE) });
```

### P.flagged

Conditional CoderType that encodes/decodes a value only if a flag is present.

**Returns**: CoderType representing the conditional value.

| Param | Description                                               |
| ----- | --------------------------------------------------------- |
| path  | Path to the flag value or a CoderType for the flag.       |
| inner | Inner CoderType for the value.                            |
| def   | Optional default value to use if the flag is not present. |

### P.optional

Optional CoderType that encodes/decodes a value based on a flag.

**Returns**: CoderType representing the optional value.

| Param | Description                                               |
| ----- | --------------------------------------------------------- |
| flag  | CoderType for the flag value.                             |
| inner | Inner CoderType for the value.                            |
| def   | Optional default value to use if the flag is not present. |

```js
// Will decode into P.U32BE only if flag present
const optional = P.optional(P.flag(new Uint8Array([0x0, 0x1])), P.U32BE);
```

```js
// If no flag present, will decode into default value
const optionalWithDefault = P.optional(P.flag(new Uint8Array([0x0, 0x1])), P.U32BE, 123);
```

```js
const s = P.struct({
  f: P.flag(new Uint8Array([0x0, 0x1])),
  f2: P.flagged('f', P.U32BE),
});
```

```js
const s2 = P.struct({
  f: P.flag(new Uint8Array([0x0, 0x1])),
  f2: P.flagged('f', P.U32BE, 123),
});
```

## Wrappers

### P.apply

Applies a base coder to a CoderType.

**Returns**: CoderType representing the transformed value.

| Param | Description              |
| ----- | ------------------------ |
| inner | The inner CoderType.     |
| b     | The base coder to apply. |

```js
import { hex } from '@scure/base';
const hex = P.apply(P.bytes(32), hex); // will decode bytes into a hex string
```

### P.wrap

Wraps a stream encoder into a generic encoder and optionally validation function

| Param | Description                                    |
| ----- | ---------------------------------------------- |
| inner | BytesCoderStream & { validate?: Validate<T> }. |

```js
const U8 = P.wrap({
  encodeStream: (w: Writer, value: number) => w.byte(value),
  decodeStream: (r: Reader): number => r.byte()
});

const checkedU8 = P.wrap({
  encodeStream: (w: Writer, value: number) => w.byte(value),
  decodeStream: (r: Reader): number => r.byte()
  validate: (n: number) => {
   if (n > 10) throw new Error(`${n} > 10`);
   return n;
  }
});
```

### P.lazy

Lazy CoderType that is evaluated at runtime.

**Returns**: CoderType representing the lazy value.

| Param | Description                            |
| ----- | -------------------------------------- |
| fn    | A function that returns the CoderType. |

```js
type Tree = { name: string; children: Tree[] };
const tree = P.struct({
  name: P.cstring,
  children: P.array(
    P.U16BE,
    P.lazy((): P.CoderType<Tree> => tree)
  ),
});
```

## Bit fiddling

Bit fiddling is implementing using primitive called Bitset: a small structure to store position of ranges that have been read.
Can be more efficient when internal trees are utilized at the cost of complexity.
Needs `O(N/8)` memory for parsing.
Purpose: if there are pointers in parsed structure,
they can cause read of two distinct ranges:
[0-32, 64-128], which means 'pos' is not enough to handle them

### P.bits

CoderType for parsing individual bits.
NOTE: Structure should parse whole amount of bytes before it can start parsing byte-level elements.

**Returns**: CoderType representing the parsed bits.

| Param | Description              |
| ----- | ------------------------ |
| len   | Number of bits to parse. |

```js
const s = P.struct({ magic: P.bits(1), version: P.bits(1), tag: P.bits(4), len: P.bits(2) });
```

### P.bitset

Bitset of boolean values with optional padding.

**Returns**: CoderType representing the bitset.

| Param | Description                                        |
| ----- | -------------------------------------------------- |
| names | An array of string names for the bitset values.    |
| pad   | Whether to pad the bitset to a multiple of 8 bits. |

```js
const myBitset = P.bitset(['flag1', 'flag2', 'flag3', 'flag4'], true);
```

## utils

#### P.validate

Validates a value before encoding and after decoding using a provided function.

**Returns**: CoderType which check value with validation function.

| Param | Description              |
| ----- | ------------------------ |
| inner | The inner CoderType.     |
| fn    | The validation function. |

```js
const val = (n: number) => {
  if (n > 10) throw new Error(`${n} > 10`);
  return n;
};

const RangedInt = P.validate(P.U32LE, val); // Will check if value is <= 10 during encoding and decoding
```

#### coders.dict

Base coder for working with dictionaries (records, objects, key-value map)
Dictionary is dynamic type like: `[key: string, value: any][]`

**Returns**: base coder that encodes/decodes between arrays of key-value tuples and dictionaries.

```js
const dict: P.CoderType<Record<string, number>> = P.apply(
 P.array(P.U16BE, P.tuple([P.cstring, P.U32LE] as const)),
 P.coders.dict()
);
```

#### coders.decimal

Base coder for working with decimal numbers.

**Returns**: base coder that encodes/decodes between bigints and decimal strings.

| Param     | Default            | Description                                                            |
| --------- | ------------------ | ---------------------------------------------------------------------- |
| precision |                    | Number of decimal places.                                              |
| round     | <code>false</code> | Round fraction part if bigger than precision (throws error by default) |

```js
const decimal8 = P.coders.decimal(8);
decimal8.encode(630880845n); // '6.30880845'
decimal8.decode('6.30880845'); // 630880845n
```

#### coders.match

Combines multiple coders into a single coder, allowing conditional encoding/decoding based on input.
Acts as a parser combinator, splitting complex conditional coders into smaller parts.

`encode = [Ae, Be]; decode = [Ad, Bd]`
->
`match([{encode: Ae, decode: Ad}, {encode: Be; decode: Bd}])`

**Returns**: Combined coder for conditional encoding/decoding.

| Param | Description               |
| ----- | ------------------------- |
| lst   | Array of coders to match. |

#### coders.reverse

Reverses direction of coder

## Debugger

There is a second optional module for debugging into console.

```ts
import * as P from 'micro-packed';
import * as PD from 'micro-packed/debugger';

const debugInt = PD.debug(P.U32LE); // Will print info to console
// PD.decode(<coder>, data);
// PD.diff(<coder>, actual, expected);
```

![Decode](./test/screens/decode.png)

![Diff](./test/screens/diff.png)

## License

MIT (c) Paul Miller [(https://paulmillr.com)](https://paulmillr.com), see LICENSE file.
