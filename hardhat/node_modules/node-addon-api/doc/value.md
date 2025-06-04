# Value

`Napi::Value` is the C++ manifestation of a JavaScript value.

Value is a the base class upon which other JavaScript values such as Number, Boolean, String, and Object are based.

The following classes inherit, either directly or indirectly, from `Napi::Value`:

- [`Napi::Array`](array.md)
- [`Napi::ArrayBuffer`](array_buffer.md)
- [`Napi::Boolean`](boolean.md)
- [`Napi::Buffer`](buffer.md)
- [`Napi::Date`](date.md)
- [`Napi::External`](external.md)
- [`Napi::Function`](function.md)
- [`Napi::Name`](name.md)
- [`Napi::Number`](number.md)
- [`Napi::Object`](object.md)
- [`Napi::String`](string.md)
- [`Napi::Symbol`](symbol.md)
- [`Napi::TypedArray`](typed_array.md)
- [`Napi::TypedArrayOf`](typed_array_of.md)

## Methods

### Empty Constructor

```cpp
Napi::Value::Value();
```

Creates a new *empty* `Napi::Value` instance.

### Constructor

```cpp
Napi::Value::Value(napi_env env, napi_value value);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Value` object.

- `[in] value`: The C++ primitive from which to instantiate the `Napi::Value`. `value` may be any of:
  - `bool`
  - Any integer type
  - Any floating point type
  - `const char*` (encoded using UTF-8, null-terminated)
  - `const char16_t*` (encoded using UTF-16-LE, null-terminated)
  - `std::string` (encoded using UTF-8)
  - `std::u16string`
  - `Napi::Value`
  - `napi_value`

### From

```cpp
template <typename T> static Napi::Value Napi::Value::From(napi_env env, const T& value);
```

- `[in] env`: The `napi_env` environment in which to create the `Napi::Value` object.

- `[in] value`: The N-API primitive value from which to create the `Napi::Value` object.

Returns a `Napi::Value` object from an N-API primitive value.

### operator napi_value

```cpp
operator napi_value() const;
```

Returns this Value's N-API value primitive.

Returns `nullptr` if this `Napi::Value` is *empty*.

### operator ==

```cpp

bool Napi::Value::operator ==(const Napi::Value& other) const;
```

- `[in] other`: The `Napi::Value` object to be compared.

Returns a `bool` indicating if this `Napi::Value` strictly equals another `Napi::Value`.

### operator !=

```cpp
bool Napi::Value::operator !=(const Napi::Value& other) const;
```

- `[in] other`: The `Napi::Value` object to be compared.

Returns a `bool` indicating if this `Napi::Value` does not strictly equal another `Napi::Value`.

### StrictEquals

```cpp
bool Napi::Value::StrictEquals(const Napi::Value& other) const;
```
- `[in] other`: The `Napi::Value` object to be compared.

Returns a `bool` indicating if this `Napi::Value` strictly equals another `Napi::Value`.

### Env

```cpp
Napi::Env Napi::Value::Env() const;
```

Returns the `Napi::Env` environment this value is associated with.

### IsEmpty

```cpp
bool Napi::Value::IsEmpty() const;
```

Returns a `bool` indicating if this `Napi::Value` is *empty* (uninitialized).

An empty `Napi::Value` is invalid, and most attempts to perform an operation on an empty Value will result in an exception.
Note an empty `Napi::Value` is distinct from JavaScript `null` or `undefined`, which are valid values.

When C++ exceptions are disabled at compile time, a method with a `Napi::Value` return type may return an empty Value to indicate a pending exception. So when not using C++ exceptions, callers should check whether this `Napi::Value` is empty before attempting to use it.

### Type

```cpp
napi_valuetype Napi::Value::Type() const;
```

Returns the `napi_valuetype` type of the `Napi::Value`.

### IsUndefined

```cpp
bool Napi::Value::IsUndefined() const;
```

Returns a `bool` indicating if this `Napi::Value` is an undefined JavaScript value.

### IsNull

```cpp
bool Napi::Value::IsNull() const;
```

Returns a `bool` indicating if this `Napi::Value` is a null JavaScript value.

### IsBoolean

```cpp
bool Napi::Value::IsBoolean() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript boolean.

### IsNumber

```cpp
bool Napi::Value::IsNumber() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript number.

### IsString

```cpp
bool Napi::Value::IsString() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript string.

### IsSymbol

```cpp
bool Napi::Value::IsSymbol() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript symbol.

### IsArray

```cpp
bool Napi::Value::IsArray() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript array.

### IsArrayBuffer

```cpp
bool Napi::Value::IsArrayBuffer() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript array buffer.

### IsTypedArray

```cpp
bool Napi::Value::IsTypedArray() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript typed array.

### IsObject

```cpp
bool Napi::Value::IsObject() const;
```

Returns a `bool` indicating if this `Napi::Value` is JavaScript object.

### IsFunction

```cpp
bool Napi::Value::IsFunction() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript function.

### IsBuffer

```cpp
bool Napi::Value::IsBuffer() const;
```

Returns a `bool` indicating if this `Napi::Value` is a Node buffer.

### IsDate

```cpp
bool Napi::Value::IsDate() const;
```

Returns a `bool` indicating if this `Napi::Value` is a JavaScript date.

### As

```cpp
template <typename T> T Napi::Value::As() const;
```

Casts to another type of `Napi::Value`, when the actual type is known or assumed.

This conversion does not coerce the type. Calling any methods inappropriate for the actual value type will throw `Napi::Error`.

### ToBoolean

```cpp
Napi::Boolean Napi::Value::ToBoolean() const;
```

Returns the `Napi::Value` coerced to a JavaScript boolean.

### ToNumber

```cpp
Napi::Number Napi::Value::ToNumber() const;
```

Returns the `Napi::Value` coerced to a JavaScript number.

### ToString

```cpp
Napi::String Napi::Value::ToString() const;
```

Returns the `Napi::Value` coerced to a JavaScript string.

### ToObject

```cpp
Napi::Object Napi::Value::ToObject() const;
```

Returns the `Napi::Value` coerced to a JavaScript object.
