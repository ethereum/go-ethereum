# Error

The `Napi::Error` class is a representation of the JavaScript `Error` object that is thrown
when runtime errors occur. The Error object can also be used as a base object for
user-defined exceptions.

The `Napi::Error` class is a persistent reference to a JavaScript error object thus
inherits its behavior from the `Napi::ObjectReference` class (for more info see: [`Napi::ObjectReference`](object_reference.md)).

If C++ exceptions are enabled (for more info see: [Setup](setup.md)), then the
`Napi::Error` class extends `std::exception` and enables integrated
error-handling for C++ exceptions and JavaScript exceptions.

For more details about error handling refer to the section titled [Error handling](error_handling.md).

## Methods

### New

Creates empty instance of an `Napi::Error` object for the specified environment.

```cpp
Napi::Error::New(Napi::Env env);
```

- `[in] env`: The environment in which to construct the `Napi::Error` object.

Returns an instance of `Napi::Error` object.

### New

Creates instance of an `Napi::Error` object.

```cpp
Napi::Error::New(Napi::Env env, const char* message);
```

- `[in] env`: The environment in which to construct the `Napi::Error` object.
- `[in] message`: Null-terminated string to be used as the message for the `Napi::Error`.

Returns instance of an `Napi::Error` object.

### New

Creates instance of an `Napi::Error` object

```cpp
Napi::Error::New(Napi::Env env, const std::string& message);
```

- `[in] env`: The environment in which to construct the `Napi::Error` object.
- `[in] message`: Reference string to be used as the message for the `Napi::Error`.

Returns instance of an `Napi::Error` object.

### Fatal

In case of an unrecoverable error in a native module, a fatal error can be thrown
to immediately terminate the process.

```cpp
static NAPI_NO_RETURN void Napi::Error::Fatal(const char* location, const char* message);
```

The function call does not return, the process will be terminated.

### Constructor

Creates empty instance of an `Napi::Error`.

```cpp
Napi::Error::Error();
```

Returns an instance of `Napi::Error` object.

### Constructor

Initializes an `Napi::Error` instance from an existing JavaScript error object.

```cpp
Napi::Error::Error(napi_env env, napi_value value);
```

- `[in] env`: The environment in which to construct the error object.
- `[in] value`: The `Napi::Error` reference to wrap.

Returns instance of an `Napi::Error` object.

### Message

```cpp
std::string& Napi::Error::Message() const NAPI_NOEXCEPT;
```

Returns the reference to the string that represent the message of the error.

### ThrowAsJavaScriptException

Throw the error as JavaScript exception.

```cpp
void Napi::Error::ThrowAsJavaScriptException() const;
```

Throws the error as a JavaScript exception.

### what

```cpp
const char* Napi::Error::what() const NAPI_NOEXCEPT override;
```

Returns a pointer to a null-terminated string that is used to identify the
exception. This method can be used only if the exception mechanism is enabled.
