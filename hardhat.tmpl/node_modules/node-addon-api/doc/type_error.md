# TypeError

The `Napi::TypeError` class is a representation of the JavaScript `TypeError` that is
thrown when an operand or argument passed to a function is incompatible with the
type expected by the operator or function.

The `Napi::TypeError` class inherits its behaviors from the `Napi::Error` class (for more info
see: [`Napi::Error`](error.md)).

For more details about error handling refer to the section titled [Error handling](error_handling.md).

## Methods

### New

Creates a new instance of the `Napi::TypeError` object.

```cpp
Napi::TypeError::New(Napi:Env env, const char* message);
```

- `[in] Env`: The environment in which to construct the `Napi::TypeError` object.
- `[in] message`: Null-terminated string to be used as the message for the `Napi::TypeError`.

Returns an instance of a `Napi::TypeError` object.

### New

Creates a new instance of a `Napi::TypeError` object.

```cpp
Napi::TypeError::New(Napi:Env env, const std::string& message);
```

- `[in] Env`: The environment in which to construct the `Napi::TypeError` object.
- `[in] message`: Reference string to be used as the message for the `Napi::TypeError`.

Returns an instance of a `Napi::TypeError` object.

### Constructor

Creates a new empty instance of a `Napi::TypeError`.

```cpp
Napi::TypeError::TypeError();
```

### Constructor

Initializes a `Napi::TypeError` instance from an existing JavaScript error object.

```cpp
Napi::TypeError::TypeError(napi_env env, napi_value value);
```

- `[in] Env`: The environment in which to construct the `Napi::TypeError` object.
- `[in] value`: The `Napi::Error` reference to wrap.

Returns an instance of a `Napi::TypeError` object.
