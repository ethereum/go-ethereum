# Basic Types

Node Addon API consists of a few fundamental data types. These allow a user of
the API to create, convert and introspect fundamental JavaScript types, and
interoperate with their C++ counterparts.

## Value

`Napi::Value` is the base class of Node Addon API's fundamental object type hierarchy.
It represents a JavaScript value of an unknown type. It is a thin wrapper around
the N-API datatype `napi_value`. Methods on this class can be used to check
the JavaScript type of the underlying N-API `napi_value` and also to convert to
C++ types.

### Constructor

```cpp
Napi::Value::Value();
```

Used to create a Node Addon API `Napi::Value` that represents an **empty** value.

```cpp
Napi::Value::Value(napi_env env, napi_value value);
```

- `[in] env` - The `napi_env` environment in which to construct the `Napi::Value`
object.
- `[in] value` - The underlying JavaScript value that the `Napi::Value` instance
represents.

Returns a Node.js Addon API `Napi::Value` that represents the `napi_value` passed
in.

### Operators

#### operator napi_value

```cpp
Napi::Value::operator napi_value() const;
```

Returns the underlying N-API `napi_value`. If the instance is _empty_, this
returns `nullptr`.

#### operator ==

```cpp
bool Napi::Value::operator ==(const Value& other) const;
```

Returns `true` if this value strictly equals another value, or `false` otherwise.

#### operator !=

```cpp
bool Napi::Value::operator !=(const Value& other) const;
```

Returns `false` if this value strictly equals another value, or `true` otherwise.

### Methods

#### From
```cpp
template <typename T>
static Napi::Value Napi::Value::From(napi_env env, const T& value);
```

- `[in] env` - The `napi_env` environment in which to construct the `Napi::Value` object.
- `[in] value` - The C++ type to represent in JavaScript.

Returns a `Napi::Value` representing the input C++ type in JavaScript.

This method is used to convert from a C++ type to a JavaScript value.
Here, `value` may be any of:
- `bool` - returns a `Napi::Boolean`.
- Any integer type - returns a `Napi::Number`.
- Any floating point type - returns a `Napi::Number`.
- `const char*` (encoded using UTF-8, null-terminated) - returns a `Napi::String`.
- `const char16_t*` (encoded using UTF-16-LE, null-terminated) - returns a `Napi::String`.
- `std::string` (encoded using UTF-8) - returns a `Napi::String`.
- `std::u16string` - returns a `Napi::String`.
- `napi::Value` - returns a `Napi::Value`.
- `napi_value` - returns a `Napi::Value`.

#### As
```cpp
template <typename T> T Napi::Value::As() const;
```

Returns the `Napi::Value` cast to a desired C++ type.

Use this when the actual type is known or assumed.

Note:
This conversion does NOT coerce the type. Calling any methods inappropriate for
the actual value type will throw `Napi::Error`.

#### StrictEquals
```cpp
bool Napi::Value::StrictEquals(const Value& other) const;
```

- `[in] other` - The value to compare against.

Returns true if the other `Napi::Value` is strictly equal to this one.

#### Env
```cpp
Napi::Env Napi::Value::Env() const;
```

Returns the environment that the value is associated with. See
[`Napi::Env`](env.md) for more details about environments.

#### IsEmpty
```cpp
bool Napi::Value::IsEmpty() const;
```

Returns `true` if the value is uninitialized.

An empty value is invalid, and most attempts to perform an operation on an
empty value will result in an exception. An empty value is distinct from
JavaScript `null` or `undefined`, which are valid values.

When C++ exceptions are disabled at compile time, a method with a `Napi::Value`
return type may return an empty value to indicate a pending exception. If C++
exceptions are not being used, callers should check the result of
`Env::IsExceptionPending` before attempting to use the value.

#### Type
```cpp
napi_valuetype Napi::Value::Type() const;
```

Returns the underlying N-API `napi_valuetype` of the value.

#### IsUndefined
```cpp
bool Napi::Value::IsUndefined() const;
```

Returns `true` if the underlying value is a JavaScript `undefined` or `false`
otherwise.

#### IsNull
```cpp
bool Napi::Value::IsNull() const;
```

Returns `true` if the underlying value is a JavaScript `null` or `false`
otherwise.

#### IsBoolean
```cpp
bool Napi::Value::IsBoolean() const;
```

Returns `true` if the underlying value is a JavaScript `true` or JavaScript
`false`, or `false` if the value is not a `Napi::Boolean` value in JavaScript.

#### IsNumber
```cpp
bool Napi::Value::IsNumber() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Number` or `false`
otherwise.

#### IsString
```cpp
bool Napi::Value::IsString() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::String` or `false`
otherwise.

#### IsSymbol
```cpp
bool Napi::Value::IsSymbol() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Symbol` or `false`
otherwise.

#### IsArray
```cpp
bool Napi::Value::IsArray() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Array` or `false`
otherwise.

#### IsArrayBuffer
```cpp
bool Napi::Value::IsArrayBuffer() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::ArrayBuffer` or `false`
otherwise.

#### IsTypedArray
```cpp
bool Napi::Value::IsTypedArray() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::TypedArray` or `false`
otherwise.

#### IsObject
```cpp
bool Napi::Value::IsObject() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Object` or `false`
otherwise.

#### IsFunction
```cpp
bool Napi::Value::IsFunction() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Function` or `false`
otherwise.

#### IsPromise
```cpp
bool Napi::Value::IsPromise() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::Promise` or `false`
otherwise.

#### IsDataView
```cpp
bool Napi::Value::IsDataView() const;
```

Returns `true` if the underlying value is a JavaScript `Napi::DataView` or `false`
otherwise.

#### IsBuffer
```cpp
bool Napi::Value::IsBuffer() const;
```

Returns `true` if the underlying value is a Node.js `Napi::Buffer` or `false`
otherwise.

#### IsExternal
```cpp
bool Napi::Value::IsExternal() const;
```

Returns `true` if the underlying value is a N-API external object or `false`
otherwise.

#### IsDate
```cpp
bool Napi::Value::IsDate() const;
```

Returns `true` if the underlying value is a JavaScript `Date` or `false`
otherwise.

#### ToBoolean
```cpp
Napi::Boolean Napi::Value::ToBoolean() const;
```

Returns a `Napi::Boolean` representing the `Napi::Value`.

This is a wrapper around `napi_coerce_to_boolean`. This will throw a JavaScript
exception if the coercion fails. If C++ exceptions are not being used, callers
should check the result of `Env::IsExceptionPending` before attempting to use
the returned value.

#### ToNumber
```cpp
Napi::Number Napi::Value::ToNumber() const;
```

Returns a `Napi::Number` representing the `Napi::Value`.

Note:
This can cause script code to be executed according to JavaScript semantics.
This is a wrapper around `napi_coerce_to_number`. This will throw a JavaScript
exception if the coercion fails. If C++ exceptions are not being used, callers
should check the result of `Env::IsExceptionPending` before attempting to use
the returned value.

#### ToString
```cpp
Napi::String Napi::Value::ToString() const;
```

Returns a `Napi::String` representing the `Napi::Value`.

Note that this can cause script code to be executed according to JavaScript
semantics. This is a wrapper around `napi_coerce_to_string`. This will throw a
JavaScript exception if the coercion fails. If C++ exceptions are not being
used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

#### ToObject
```cpp
Napi::Object Napi::Value::ToObject() const;
```

Returns a `Napi::Object` representing the `Napi::Value`.

This is a wrapper around `napi_coerce_to_object`. This will throw a JavaScript
exception if the coercion fails. If C++ exceptions are not being used, callers
should check the result of `Env::IsExceptionPending` before attempting to use
the returned value.

## Name

Names are JavaScript values that can be used as a property name. There are two
specialized types of names supported in Node.js Addon API [`Napi::String`](string.md)
and [`Napi::Symbol`](symbol.md).

### Methods

#### Constructor
```cpp
Napi::Name::Name();
```

Returns an empty `Napi::Name`.

```cpp
Napi::Name::Name(napi_env env, napi_value value);
```
- `[in] env` - The environment in which to create the array.
- `[in] value` - The primitive to wrap.

Returns a `Napi::Name` created from the JavaScript primitive.

Note:
The value is not coerced to a string.

## Array

Arrays are native representations of JavaScript Arrays. `Napi::Array` is a wrapper
around `napi_value` representing a JavaScript Array.

[`Napi::TypedArray`][] and [`Napi::ArrayBuffer`][] correspond to JavaScript data
types such as [`Int32Array`][] and [`ArrayBuffer`][], respectively, that can be
used for transferring large amounts of data from JavaScript to the native side.
An example illustrating the use of a JavaScript-provided `ArrayBuffer` in native
code is available [here](https://github.com/nodejs/node-addon-examples/tree/master/array_buffer_to_native/node-addon-api).

### Constructor
```cpp
Napi::Array::Array();
```

Returns an empty array.

If an error occurs, a `Napi::Error` will be thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

```cpp
Napi::Array::Array(napi_env env, napi_value value);
```
- `[in] env` - The environment in which to create the array.
- `[in] value` - The primitive to wrap.

Returns a `Napi::Array` wrapping a `napi_value`.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

### Methods

#### New
```cpp
static Napi::Array Napi::Array::New(napi_env env);
```
- `[in] env` - The environment in which to create the array.

Returns a new `Napi::Array`.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

#### New

```cpp
static Napi::Array Napi::Array::New(napi_env env, size_t length);
```
- `[in] env` - The environment in which to create the array.
- `[in] length` - The length of the array.

Returns a new `Napi::Array` with the given length.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

#### Length
```cpp
uint32_t Napi::Array::Length() const;
```

Returns the length of the array.

Note:
This can execute JavaScript code implicitly according to JavaScript semantics.
If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

[`Napi::TypedArray`]: ./typed_array.md
[`Napi::ArrayBuffer`]: ./array_buffer.md
[`Int32Array`]: https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Int32Array
[`ArrayBuffer`]: https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/ArrayBuffer
