# Boolean

`Napi::Boolean` class is a representation of the JavaScript `Boolean` object. The
`Napi::Boolean` class inherits its behavior from the `Napi::Value` class
(for more info see: [`Napi::Value`](value.md)).

## Methods

### Constructor

Creates a new empty instance of an `Napi::Boolean` object.

```cpp
Napi::Boolean::Boolean();
```

Returns a new _empty_  `Napi::Boolean` object.

### Contructor

Creates a new instance of the `Napi::Boolean` object.

```cpp
Napi::Boolean(napi_env env, napi_value value);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Boolean` object.
- `[in] value`: The `napi_value` which is a handle for a JavaScript `Boolean`.

Returns a non-empty `Napi::Boolean` object.

### New

Initializes a new instance of the `Napi::Boolean` object.

```cpp
Napi::Boolean Napi::Boolean::New(napi_env env, bool value);
```
- `[in] env`: The `napi_env` environment in which to construct the `Napi::Boolean` object.
- `[in] value`: The primitive boolean value (`true` or `false`).

Returns a new instance of the `Napi::Boolean` object.

### Value

Converts a `Napi::Boolean` value to a boolean primitive.

```cpp
bool Napi::Boolean::Value() const;
```

Returns the boolean primitive type of the corresponding `Napi::Boolean` object.

## Operators

### operator bool

Converts a `Napi::Boolean` value to a boolean primitive.

```cpp
Napi::Boolean::operator bool() const;
```

Returns the boolean primitive type of the corresponding `Napi::Boolean` object.
