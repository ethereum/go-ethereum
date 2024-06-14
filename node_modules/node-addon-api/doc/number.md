# Number

`Napi::Number` class is a representation of the JavaScript `Number` object. The
`Napi::Number` class inherits its behavior from `Napi::Value` class
(for more info see [`Napi::Value`](value.md))

## Methods

### Constructor

Creates a new _empty_ instance of a `Napi::Number` object.

```cpp
Napi::Number();
```

Returns a new _empty_ `Napi::Number` object.

### Contructor

Creates a new instance of a `Napi::Number` object.

```cpp
Napi::Number(napi_env env, napi_value value);
```

 - `[in] env`: The `napi_env` environment in which to construct the `Napi::Number` object.
 - `[in] value`: The JavaScript value holding a number.

 Returns a non-empty `Napi::Number` object.

 ### New

 Creates a new instance of a `Napi::Number` object.

```cpp
Napi::Number Napi::Number::New(napi_env env, double value);
```
 - `[in] env`: The `napi_env` environment in which to construct the `Napi::Number` object.
 - `[in] value`: The C++ primitive from which to instantiate the `Napi::Number`.

Creates a new instance of a `Napi::Number` object.

### Int32Value

Converts a `Napi::Number` value to a `int32_t` primitive type.

```cpp
Napi::Number::Int32Value() const;
```

Returns the `int32_t` primitive type of the corresponding `Napi::Number` object.

### Uint32Value

Converts a `Napi::Number` value to a `uint32_t` primitive type.

```cpp
Napi::Number::Uint32Value() const;
```

Returns the `uint32_t` primitive type of the corresponding `Napi::Number` object.

### Int64Value

Converts a `Napi::Number` value to a `int64_t` primitive type.

```cpp
Napi::Number::Int64Value() const;
```

Returns the `int64_t` primitive type of the corresponding `Napi::Number` object.

### FloatValue

Converts a `Napi::Number` value to a `float` primitive type.

```cpp
Napi::Number::FloatValue() const;
```

Returns the `float` primitive type of the corresponding `Napi::Number` object.

### DoubleValue

Converts a `Napi::Number` value to a `double` primitive type.

```cpp
Napi::Number::DoubleValue() const;
```

Returns the `double` primitive type of the corresponding `Napi::Number` object.

## Operators

The `Napi::Number` class contains a set of operators to easily cast JavaScript
`Number` object to one of the following primitive types:

 - `int32_t`
 - `uint32_t`
 - `int64_t`
 - `float`
 - `double`

### operator int32_t

Converts a `Napi::Number` value to a `int32_t` primitive.

```cpp
Napi::Number::operator int32_t() const;
```

Returns the `int32_t` primitive type of the corresponding `Napi::Number` object.

### operator uint32_t

Converts a `Napi::Number` value to a `uint32_t` primitive type.

```cpp
Napi::Number::operator uint32_t() const;
```

Returns the `uint32_t` primitive type of the corresponding `Napi::Number` object.

### operator int64_t

Converts a `Napi::Number` value to a `int64_t` primitive type.

```cpp
Napi::Number::operator int64_t() const;
```

Returns the `int64_t` primitive type of the corresponding `Napi::Number` object.

### operator float

Converts a `Napi::Number` value to a `float` primitive type.

```cpp
Napi::Number::operator float() const;
```

Returns the `float` primitive type of the corresponding `Napi::Number` object.

### operator double

Converts a `Napi::Number` value to a `double` primitive type.

```cpp
Napi::Number::operator double() const;
```

Returns the `double` primitive type of the corresponding `Napi::Number` object.

### Example

The following shows an example of casting a number to an `uint32_t` value.

```cpp
uint32_t operatorVal = Napi::Number::New(Env(), 10.0); // Number to unsigned 32 bit integer
// or
auto instanceVal = info[0].As<Napi::Number>().Uint32Value();
```
