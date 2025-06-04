# Date

`Napi::Date` class is a representation of the JavaScript `Date` object. The
`Napi::Date` class inherits its behavior from `Napi::Value` class
(for more info see [`Napi::Value`](value.md))

## Methods

### Constructor

Creates a new _empty_ instance of a `Napi::Date` object.

```cpp
Napi::Date::Date();
```

Creates a new _non-empty_ instance of a `Napi::Date` object.

```cpp
Napi::Date::Date(napi_env env, napi_value value);
```

 - `[in] env`: The environment in which to construct the `Napi::Date` object.
 - `[in] value`: The `napi_value` which is a handle for a JavaScript `Date`.

### New

Creates a new instance of a `Napi::Date` object.

```cpp
static Napi::Date Napi::Date::New(Napi::Env env, double value);
```

 - `[in] env`: The environment in which to construct the `Napi::Date` object.
 - `[in] value`: The time value the JavaScript `Date` will contain represented
  as the number of milliseconds since 1 January 1970 00:00:00 UTC.

Returns a new instance of `Napi::Date` object.

### ValueOf

```cpp
double Napi::Date::ValueOf() const;
```

Returns the time value as `double` primitive represented as the number of
 milliseconds since 1 January 1970 00:00:00 UTC.

## Operators

### operator double

Converts a `Napi::Date` value to a `double` primitive.

```cpp
Napi::Date::operator double() const;
```

### Example

The following shows an example of casting a `Napi::Date` value to a `double`
 primitive.

```cpp
double operatorVal = Napi::Date::New(Env(), 0); // Napi::Date to double
// or
auto instanceVal = info[0].As<Napi::Date>().ValueOf();
```
