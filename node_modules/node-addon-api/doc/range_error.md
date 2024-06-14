# RangeError

The `Napi::RangeError` class is a representation of the JavaScript `RangeError` that is
thrown when trying to pass a value as an argument to a function that does not allow
a range that includes the value.

The `Napi::RangeError` class inherits its behaviors from the `Napi::Error` class (for
more info see: [`Napi::Error`](error.md)).

For more details about error handling refer to the section titled [Error handling](error_handling.md).

## Methods

### New

Creates a new instance of a `Napi::RangeError` object.

```cpp
Napi::RangeError::New(Napi::Env env, const char* message);
```

- `[in] Env`: The environment in which to construct the `Napi::RangeError` object.
- `[in] message`: Null-terminated string to be used as the message for the `Napi::RangeError`.

Returns an instance of a `Napi::RangeError` object.

### New

Creates a new instance of a `Napi::RangeError` object.

```cpp
Napi::RangeError::New(Napi::Env env, const std::string& message);
```

- `[in] Env`: The environment in which to construct the `Napi::RangeError` object.
- `[in] message`: Reference string to be used as the message for the `Napi::RangeError`.

Returns an instance of a `Napi::RangeError` object.

### Constructor

Creates a new empty instance of a `Napi::RangeError`.

```cpp
Napi::RangeError::RangeError();
```

### Constructor

Initializes a `Napi::RangeError` instance from an existing Javascript error object.

```cpp
Napi::RangeError::RangeError(napi_env env, napi_value value);
```

- `[in] Env`: The environment in which to construct the `Napi::RangeError` object.
- `[in] value`: The `Napi::Error` reference to wrap.

Returns an instance of a `Napi::RangeError` object.
