# DataView

The `Napi::DataView` class corresponds to the
[JavaScript `DataView`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/DataView)
class.

## Methods

### New

Allocates a new `Napi::DataView` instance with a given `Napi::ArrayBuffer`.

```cpp
static Napi::DataView Napi::DataView::New(napi_env env, Napi::ArrayBuffer arrayBuffer);
```

- `[in] env`: The environment in which to create the `Napi::DataView` instance.
- `[in] arrayBuffer` : `Napi::ArrayBuffer` underlying the `Napi::DataView`.

Returns a new `Napi::DataView` instance.

### New

Allocates a new `Napi::DataView` instance with a given `Napi::ArrayBuffer`.

```cpp
static Napi::DataView Napi::DataView::New(napi_env env, Napi::ArrayBuffer arrayBuffer, size_t byteOffset);
```

- `[in] env`: The environment in which to create the `Napi::DataView` instance.
- `[in] arrayBuffer` : `Napi::ArrayBuffer` underlying the `Napi::DataView`.
- `[in] byteOffset` : The byte offset within the `Napi::ArrayBuffer` from which to start projecting the `Napi::DataView`.

Returns a new `Napi::DataView` instance.

### New

Allocates a new `Napi::DataView` instance with a given `Napi::ArrayBuffer`.

```cpp
static Napi::DataView Napi::DataView::New(napi_env env, Napi::ArrayBuffer arrayBuffer, size_t byteOffset, size_t byteLength);
```

- `[in] env`: The environment in which to create the `Napi::DataView` instance.
- `[in] arrayBuffer` : `Napi::ArrayBuffer` underlying the `Napi::DataView`.
- `[in] byteOffset` : The byte offset within the `Napi::ArrayBuffer` from which to start projecting the `Napi::DataView`.
- `[in] byteLength` : Number of elements in the `Napi::DataView`.

Returns a new `Napi::DataView` instance.

### Constructor

Initializes an empty instance of the `Napi::DataView` class.

```cpp
Napi::DataView();
```

### Constructor

Initializes a wrapper instance of an existing `Napi::DataView` instance.

```cpp
Napi::DataView(napi_env env, napi_value value);
```

- `[in] env`: The environment in which to create the `Napi::DataView` instance.
- `[in] value`: The `Napi::DataView` reference to wrap.

### ArrayBuffer

```cpp
Napi::ArrayBuffer Napi::DataView::ArrayBuffer() const;
```

Returns the backing array buffer.

### ByteOffset

```cpp
size_t Napi::DataView::ByteOffset() const;
```

Returns the offset into the `Napi::DataView` where the array starts, in bytes.

### ByteLength

```cpp
size_t Napi::DataView::ByteLength() const;
```

Returns the length of the array, in bytes.

### GetFloat32

```cpp
float Napi::DataView::GetFloat32(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a signed 32-bit float (float) at the specified byte offset from the start of the `Napi::DataView`.

### GetFloat64

```cpp
double Napi::DataView::GetFloat64(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a signed 64-bit float (double) at the specified byte offset from the start of the `Napi::DataView`.

### GetInt8

```cpp
int8_t Napi::DataView::GetInt8(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a signed 8-bit integer (byte) at the specified byte offset from the start of the `Napi::DataView`.

### GetInt16

```cpp
int16_t Napi::DataView::GetInt16(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a signed 16-bit integer (short) at the specified byte offset from the start of the `Napi::DataView`.

### GetInt32

```cpp
int32_t Napi::DataView::GetInt32(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a signed 32-bit integer (long) at the specified byte offset from the start of the `Napi::DataView`.

### GetUint8

```cpp
uint8_t Napi::DataView::GetUint8(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a unsigned 8-bit integer (unsigned byte) at the specified byte offset from the start of the `Napi::DataView`.

### GetUint16

```cpp
uint16_t Napi::DataView::GetUint16(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a unsigned 16-bit integer (unsigned short) at the specified byte offset from the start of the `Napi::DataView`.

### GetUint32

```cpp
uint32_t Napi::DataView::GetUint32(size_t byteOffset) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.

Returns a unsigned 32-bit integer (unsigned long) at the specified byte offset from the start of the `Napi::DataView`.

### SetFloat32

```cpp
void Napi::DataView::SetFloat32(size_t byteOffset, float value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetFloat64

```cpp
void Napi::DataView::SetFloat64(size_t byteOffset, double value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetInt8

```cpp
void Napi::DataView::SetInt8(size_t byteOffset, int8_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetInt16

```cpp
void Napi::DataView::SetInt16(size_t byteOffset, int16_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetInt32

```cpp
void Napi::DataView::SetInt32(size_t byteOffset, int32_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetUint8

```cpp
void Napi::DataView::SetUint8(size_t byteOffset, uint8_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetUint16

```cpp
void Napi::DataView::SetUint16(size_t byteOffset, uint16_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.

### SetUint32

```cpp
void Napi::DataView::SetUint32(size_t byteOffset, uint32_t value) const;
```

- `[in] byteOffset`: The offset, in byte, from the start of the view where to read the data.
- `[in] value`: The value to set.
