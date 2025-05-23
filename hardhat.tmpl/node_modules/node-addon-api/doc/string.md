# String

## Constructor

```cpp
Napi::String::String();
```

Returns a new **empty** `Napi::String` instance.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

```cpp
Napi::String::String(napi_env env, napi_value value); ///< Wraps a N-API value primitive.
```
- `[in] env` - The environment in which to create the string.
- `[in] value` - The primitive to wrap.

Returns a `Napi::String` wrapping a `napi_value`.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

## Operators

### operator std::string

```cpp
Napi::String::operator std::string() const;
```

Returns a UTF-8 encoded C++ string.

### operator std::u16string
```cpp
Napi::String::operator std::u16string() const;
```

Returns a UTF-16 encoded C++ string.

## Methods

### New
```cpp
Napi::String::New();
```

Returns a new empty `Napi::String`.

### New
```cpp
Napi::String::New(napi_env env, const std::string& value);
Napi::String::New(napi_env env, const std::u16::string& value);
Napi::String::New(napi_env env, const char* value);
Napi::String::New(napi_env env, const char16_t* value);
Napi::String::New(napi_env env, const char* value, size_t length);
Napi::String::New(napi_env env, const char16_t* value, size_t length);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Value` object.
- `[in] value`: The C++ primitive from which to instantiate the `Napi::Value`. `value` may be any of:
  - `std::string&` - represents an ANSI string.
  - `std::u16string&` - represents a UTF16-LE string.
  - `const char*` - represents a UTF8 string.
  - `const char16_t*` - represents a UTF16-LE string.
- `[in] length`: The length of the string (not necessarily null-terminated) in code units.

Returns a new `Napi::String` that represents the passed in C++ string.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Env::IsExceptionPending` before
attempting to use the returned value.

### Utf8Value
```cpp
std::string Napi::String::Utf8Value() const;
```

Returns a UTF-8 encoded C++ string.

### Utf16Value
```cpp
std::u16string Napi::String::Utf16Value() const;
```

Returns a UTF-16 encoded C++ string.
