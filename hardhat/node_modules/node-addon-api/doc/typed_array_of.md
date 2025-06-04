# TypedArrayOf

The `Napi::TypedArrayOf` class corresponds to the various
[JavaScript `TypedArray`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray)
classes.

## Typedefs

The common JavaScript `TypedArray` types are pre-defined for each of use:

```cpp
typedef Napi::TypedArrayOf<int8_t> Int8Array;
typedef Napi::TypedArrayOf<uint8_t> Uint8Array;
typedef Napi::TypedArrayOf<int16_t> Int16Array;
typedef Napi::TypedArrayOf<uint16_t> Uint16Array;
typedef Napi::TypedArrayOf<int32_t> Int32Array;
typedef Napi::TypedArrayOf<uint32_t> Uint32Array;
typedef Napi::TypedArrayOf<float> Float32Array;
typedef Napi::TypedArrayOf<double> Float64Array;
```

The one exception is the `Uint8ClampedArray` which requires explicit
initialization:

```cpp
Uint8Array::New(env, length, napi_uint8_clamped_array)
```

Note that while it's possible to create a "clamped" array the _clamping_
behavior is only applied in JavaScript.

## Methods

### New

Allocates a new `Napi::TypedArray` instance with a given length. The underlying
`Napi::ArrayBuffer` is allocated automatically to the desired number of elements.

The array type parameter can normally be omitted (because it is inferred from
the template parameter T), except when creating a "clamped" array.

```cpp
static Napi::TypedArrayOf Napi::TypedArrayOf::New(napi_env env,
                        size_t elementLength,
                        napi_typedarray_type type);
```

- `[in] env`: The environment in which to create the `Napi::TypedArrayOf` instance.
- `[in] elementLength`: The length to be allocated, in elements.
- `[in] type`: The type of array to allocate (optional).

Returns a new `Napi::TypedArrayOf` instance.

### New

Wraps the provided `Napi::ArrayBuffer` into a new `Napi::TypedArray` instance.

The array `type` parameter can normally be omitted (because it is inferred from
the template parameter `T`), except when creating a "clamped" array.

```cpp
static Napi::TypedArrayOf Napi::TypedArrayOf::New(napi_env env,
                        size_t elementLength,
                        Napi::ArrayBuffer arrayBuffer,
                        size_t bufferOffset,
                        napi_typedarray_type type);
```

- `[in] env`: The environment in which to create the `Napi::TypedArrayOf` instance.
- `[in] elementLength`: The length to array, in elements.
- `[in] arrayBuffer`: The backing `Napi::ArrayBuffer` instance.
- `[in] bufferOffset`: The offset into the `Napi::ArrayBuffer` where the array starts,
                       in bytes.
- `[in] type`: The type of array to allocate (optional).

Returns a new `Napi::TypedArrayOf` instance.

### Constructor

Initializes an empty instance of the `Napi::TypedArrayOf` class.

```cpp
Napi::TypedArrayOf::TypedArrayOf();
```

### Constructor

Initializes a wrapper instance of an existing `Napi::TypedArrayOf` object.

```cpp
Napi::TypedArrayOf::TypedArrayOf(napi_env env, napi_value value);
```

- `[in] env`: The environment in which to create the `Napi::TypedArrayOf` object.
- `[in] value`: The `Napi::TypedArrayOf` reference to wrap.

### operator []

```cpp
T& Napi::TypedArrayOf::operator [](size_t index);
```

- `[in] index: The element index into the array.

Returns the element found at the given index.

### operator []

```cpp
const T& Napi::TypedArrayOf::operator [](size_t index) const;
```

- `[in] index: The element index into the array.

Returns the element found at the given index.

### Data

```cpp
T* Napi::TypedArrayOf::Data() const;
```

Returns a pointer into the backing `Napi::ArrayBuffer` which is offset to point to the
start of the array.

### Data

```cpp
const T* Napi::TypedArrayOf::Data() const
```

Returns a pointer into the backing `Napi::ArrayBuffer` which is offset to point to the
start of the array.
