# TypedArray

The `Napi::TypedArray` class corresponds to the
[JavaScript `TypedArray`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/TypedArray)
class.

## Methods

### Constructor

Initializes an empty instance of the `Napi::TypedArray` class.

```cpp
Napi::TypedArray::TypedArray();
```

### Constructor

Initializes a wrapper instance of an existing `Napi::TypedArray` instance.

```cpp
Napi::TypedArray::TypedArray(napi_env env, napi_value value);
```

- `[in] env`: The environment in which to create the `Napi::TypedArray` instance.
- `[in] value`: The `Napi::TypedArray` reference to wrap.

### TypedArrayType

```cpp
napi_typedarray_type Napi::TypedArray::TypedArrayType() const;
```

Returns the type of this instance.

### ArrayBuffer

```cpp
Napi::ArrayBuffer Napi::TypedArray::ArrayBuffer() const;
```

Returns the backing array buffer.

### ElementSize

```cpp
uint8_t Napi::TypedArray::ElementSize() const;
```

Returns the size of one element, in bytes.

### ElementLength

```cpp
size_t Napi::TypedArray::ElementLength() const;
```

Returns the number of elements.

### ByteOffset

```cpp
size_t Napi::TypedArray::ByteOffset() const;
```

Returns the offset into the `Napi::ArrayBuffer` where the array starts, in bytes.

### ByteLength

```cpp
size_t Napi::TypedArray::ByteLength() const;
```

Returns the length of the array, in bytes.
