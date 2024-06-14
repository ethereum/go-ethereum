# Symbol

## Methods

### Constructor

Instantiates a new `Napi::Symbol` value.

```cpp
Napi::Symbol::Symbol();
```

Returns a new empty `Napi::Symbol`.

### New
```cpp
Napi::Symbol::New(napi_env env, const std::string& description);
Napi::Symbol::New(napi_env env, const char* description);
Napi::Symbol::New(napi_env env, Napi::String description);
Napi::Symbol::New(napi_env env, napi_value description);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Symbol` object.
- `[in] value`: The C++ primitive which represents the description hint for the `Napi::Symbol`.
  `description` may be any of:
  - `std::string&` - ANSI string description.
  - `const char*` - represents a UTF8 string description.
  - `String` - Node addon API String description.
  - `napi_value` - N-API `napi_value` description.

If an error occurs, a `Napi::Error` will get thrown. If C++ exceptions are not
being used, callers should check the result of `Napi::Env::IsExceptionPending` before
attempting to use the returned value.

### Utf8Value
```cpp
static Napi::Symbol Napi::Symbol::WellKnown(napi_env env, const std::string& name);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Symbol` object.
- `[in] name`: The C++ string representing the `Napi::Symbol` to retrieve.

Returns a `Napi::Symbol` representing a well-known `Symbol` from the
`Symbol` registry.
