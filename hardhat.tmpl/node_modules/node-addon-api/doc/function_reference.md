# FunctionReference

`Napi::FunctionReference` is a subclass of [`Napi::Reference`](reference.md), and
is equivalent to an instance of `Napi::Reference<Napi::Function>`. This means
that a `Napi::FunctionReference` holds a [`Napi::Function`](function.md), and a
count of the number of references to that `Napi::Function`. When the count is
greater than 0, a `Napi::FunctionReference` is not eligible for garbage collection.
This ensures that the `Function` will remain accessible, even if the original
reference to it is no longer available.
`Napi::FunctionReference` allows the referenced JavaScript function object to be
called from a native add-on with two different methods: `Call` and `MakeCallback`.
See the documentation for [`Napi::Function`](function.md) for when `Call` should
be used instead of `MakeCallback` and vice-versa.

The `Napi::FunctionReference` class inherits its behavior from the `Napi::Reference`
class (for more info see: [`Napi::Reference`](reference.md)).

## Methods

### Weak

Creates a "weak" reference to the value, in that the initial reference count is
set to 0.

```cpp
static Napi::FunctionReference Napi::Weak(const Napi::Function& value);
```

- `[in] value`: The value which is to be referenced.

Returns the newly created reference.

### Persistent

Creates a "persistent" reference to the value, in that the initial reference
count is set to 1.

```cpp
static Napi::FunctionReference Napi::Persistent(const Napi::Function& value);
```

- `[in] value`: The value which is to be referenced.

Returns the newly created reference.

### Constructor

Creates a new empty instance of `Napi::FunctionReference`.

```cpp
Napi::FunctionReference::FunctionReference();
```

### Constructor

Creates a new instance of the `Napi::FunctionReference`.

```cpp
Napi::FunctionReference::FunctionReference(napi_env env, napi_ref ref);
```

- `[in] env`: The environment in which to construct the `Napi::FunctionReference` object.
- `[in] ref`: The N-API reference to be held by the `Napi::FunctionReference`.

Returns a newly created `Napi::FunctionReference` object.

### New

Constructs a new instance by calling the constructor held by this reference.

```cpp
Napi::Object Napi::FunctionReference::New(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the contructor function.

Returns a new JavaScript object.

### New

Constructs a new instance by calling the constructor held by this reference.

```cpp
Napi::Object Napi::FunctionReference::New(const std::vector<napi_value>& args) const;
```

- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the constructor function.

Returns a new JavaScript object.

### Call

Calls a referenced Javascript function from a native add-on.

```cpp
Napi::Value Napi::FunctionReference::Call(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the referenced function.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### Call

Calls a referenced JavaScript function from a native add-on.

```cpp
Napi::Value Napi::FunctionReference::Call(const std::vector<napi_value>& args) const;
```

- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the referenced function.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### Call

Calls a referenced JavaScript function from a native add-on.

```cpp
Napi::Value Napi::FunctionReference::Call(napi_value recv, const std::initializer_list<napi_value>& args) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the referenced function.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### Call

Calls a referenced JavaScript function from a native add-on.

```cpp
Napi::Value Napi::FunctionReference::Call(napi_value recv, const std::vector<napi_value>& args) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the referenced function.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### Call

Calls a referenced JavaScript function from a native add-on.

```cpp
Napi::Value Napi::FunctionReference::Call(napi_value recv, size_t argc, const napi_value* args) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] argc`: The number of arguments passed to the referenced function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the referenced function.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.


### MakeCallback

Calls a referenced JavaScript function from a native add-on after an asynchronous
operation.

```cpp
Napi::Value Napi::FunctionReference::MakeCallback(napi_value recv, const std::initializer_list<napi_value>& args, napi_async_context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the referenced function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### MakeCallback

Calls a referenced JavaScript function from a native add-on after an asynchronous
operation.

```cpp
Napi::Value Napi::FunctionReference::MakeCallback(napi_value recv, const std::vector<napi_value>& args, napi_async_context context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the referenced function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

### MakeCallback

Calls a referenced JavaScript function from a native add-on after an asynchronous
operation.

```cpp
Napi::Value Napi::FunctionReference::MakeCallback(napi_value recv, size_t argc, const napi_value* args, napi_async_context context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the referenced function when it's called.
- `[in] argc`: The number of arguments passed to the referenced function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the referenced function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript object returned by the referenced
function.

## Operator

```cpp
Napi::Value operator ()(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of reference to JavaScript values as `napi_value`

Returns a `Napi::Value` representing the JavaScript value returned by the referenced
function.
