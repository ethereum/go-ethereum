# BigInt

A JavaScript BigInt value.

## Methods

### New

```cpp
static Napi::BigInt Napi::BigInt::New(Napi::Env env, int64_t value);
```

 - `[in] env`: The environment in which to construct the `Napi::BigInt` object.
 - `[in] value`: The value the JavaScript `BigInt` will contain

These APIs convert the C `int64_t` and `uint64_t` types to the JavaScript
`BigInt` type.

```cpp
static Napi::BigInt Napi::BigInt::New(Napi::Env env,
                  int sign_bit,
                  size_t word_count,
                  const uint64_t* words);
```

 - `[in] env`: The environment in which to construct the `Napi::BigInt` object.
 - `[in] sign_bit`: Determines if the resulting `BigInt` will be positive or negative.
 - `[in] word_count`: The length of the words array.
 - `[in] words`: An array of `uint64_t` little-endian 64-bit words.

This API converts an array of unsigned 64-bit words into a single `BigInt`
value.

The resulting `BigInt` is calculated as: (–1)<sup>`sign_bit`</sup> (`words[0]`
× (2<sup>64</sup>)<sup>0</sup> + `words[1]` × (2<sup>64</sup>)<sup>1</sup> + …)

Returns a new JavaScript `BigInt`.

### Constructor

```cpp
Napi::BigInt();
```

Returns a new empty JavaScript `Napi::BigInt`.

### Int64Value

```cpp
int64_t Napi::BitInt::Int64Value(bool* lossless) const;
```

 - `[out] lossless`: Indicates whether the `BigInt` value was converted losslessly.

Returns the C `int64_t` primitive equivalent of the given JavaScript
`BigInt`. If needed it will truncate the value, setting lossless to false.

### Uint64Value

```cpp
uint64_t Napi::BigInt::Uint64Value(bool* lossless) const;
```

 - `[out] lossless`: Indicates whether the `BigInt` value was converted
   losslessly.

Returns the C `uint64_t` primitive equivalent of the given JavaScript
`BigInt`. If needed it will truncate the value, setting lossless to false.

### WordCount

```cpp
size_t Napi::BigInt::WordCount() const;
```

Returns the number of words needed to store this `BigInt` value.

### ToWords

```cpp
void Napi::BigInt::ToWords(size_t* word_count, int* sign_bit, uint64_t* words);
```

 - `[out] sign_bit`: Integer representing if the JavaScript `BigInt` is positive
   or negative.
 - `[in/out] word_count`: Must be initialized to the length of the words array.
   Upon return, it will be set to the actual number of words that would be
   needed to store this `BigInt`.
 - `[out] words`: Pointer to a pre-allocated 64-bit word array.

Returns a single `BigInt` value into a sign bit, 64-bit little-endian array,
and the number of elements in the array.
