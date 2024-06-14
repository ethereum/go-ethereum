# Function

The `Napi::Function` class provides a set of methods for creating a function object in
native code that can later be called from JavaScript. The created function is not
automatically visible from JavaScript. Instead it needs to be part of the add-on's
module exports or be returned by one of the module's exported functions.

In addition the `Napi::Function` class also provides methods that can be used to call
functions that were created in JavaScript and passed to the native add-on.

The `Napi::Function` class inherits its behavior from the `Napi::Object` class (for more info
see: [`Napi::Object`](object.md)).

## Example

```cpp
#include <napi.h>

using namespace Napi;

Value Fn(const CallbackInfo& info) {
  Env env = info.Env();
  // ...
  return String::New(env, "Hello World");
}

Object Init(Env env, Object exports) {
  exports.Set(String::New(env, "fn"), Function::New(env, Fn));
}

NODE_API_MODULE(NODE_GYP_MODULE_NAME, Init)
```

The above code can be used from JavaScript as follows:

```js
const addon = require('./addon');
addon.fn();
```

With the `Napi::Function` class it is possible to call a JavaScript function object
from a native add-on with two different methods: `Call` and `MakeCallback`.
The API of these two methods is very similar, but they are used in different
contexts. The `MakeCallback` method is used to call from native code back into
JavaScript after returning from an [asynchronous operation](async_operations.md)
and in general in situations which don't have an existing JavaScript function on
the stack. The `Call` method is used when there is already a JavaScript function
on the stack (for example when running a native method called from JavaScript).

## Methods

### Constructor

Creates a new empty instance of `Napi::Function`.

```cpp
Napi::Function::Function();
```

### Constructor

Creates a new instance of the `Napi::Function` object.

```cpp
Napi::Function::Function(napi_env env, napi_value value);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Function` object.
- `[in] value`: The `napi_value` which is a handle for a JavaScript function.

Returns a non-empty `Napi::Function` instance.

### New

Creates an instance of a `Napi::Function` object.

```cpp
template <typename Callable>
static Napi::Function Napi::Function::New(napi_env env, Callable cb, const char* utf8name = nullptr, void* data = nullptr);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Function` object.
- `[in] cb`: Object that implements `Callable`.
- `[in] utf8name`: Null-terminated string to be used as the name of the function.
- `[in] data`: User-provided data context. This will be passed back into the
function when invoked later.

Returns an instance of a `Napi::Function` object.

### New

```cpp
template <typename Callable>
static Napi::Function Napi::Function::New(napi_env env, Callable cb, const std::string& utf8name, void* data = nullptr);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::Function` object.
- `[in] cb`: Object that implements `Callable`.
- `[in] utf8name`: String to be used as the name of the function.
- `[in] data`: User-provided data context. This will be passed back into the
function when invoked later.

Returns an instance of a `Napi::Function` object.

### New

Creates a new JavaScript value from one that represents the constructor for the
object.

```cpp
Napi::Object Napi::Function::New(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the contructor function.

Returns a new JavaScript object.

### New

Creates a new JavaScript value from one that represents the constructor for the
object.

```cpp
Napi::Object Napi::Function::New(const std::vector<napi_value>& args) const;
```

- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the constructor function.

Returns a new JavaScript object.

### New

Creates a new JavaScript value from one that represents the constructor for the
object.

```cpp
Napi::Object Napi::Function::New(size_t argc, const napi_value* args) const;
```

- `[in] argc`: The number of the arguments passed to the contructor function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the constructor function.

Returns a new JavaScript object.

### Call

Calls a Javascript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### Call

Calls a JavaScript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(const std::vector<napi_value>& args) const;
```

- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### Call

Calls a Javascript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(size_t argc, const napi_value* args) const;
```

- `[in] argc`: The number of the arguments passed to the function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### Call

Calls a Javascript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(napi_value recv, const std::initializer_list<napi_value>& args) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### Call

Calls a Javascript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(napi_value recv, const std::vector<napi_value>& args) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] args`: Vector of JavaScript values as `napi_value` representing the
arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### Call

Calls a Javascript function from a native add-on.

```cpp
Napi::Value Napi::Function::Call(napi_value recv, size_t argc, const napi_value* args) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] argc`: The number of the arguments passed to the function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the function.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### MakeCallback

Calls a Javascript function from a native add-on after an asynchronous operation.

```cpp
Napi::Value Napi::Function::MakeCallback(napi_value recv, const std::initializer_list<napi_value>& args, napi_async_context context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] args`: Initializer list of JavaScript values as `napi_value` representing
the arguments of the function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### MakeCallback

Calls a Javascript function from a native add-on after an asynchronous operation.

```cpp
Napi::Value Napi::Function::MakeCallback(napi_value recv, const std::vector<napi_value>& args, napi_async_context context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] args`: List of JavaScript values as `napi_value` representing the
arguments of the function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

### MakeCallback

Calls a Javascript function from a native add-on after an asynchronous operation.

```cpp
Napi::Value Napi::Function::MakeCallback(napi_value recv, size_t argc, const napi_value* args, napi_async_context context = nullptr) const;
```

- `[in] recv`: The `this` object passed to the called function.
- `[in] argc`: The number of the arguments passed to the function.
- `[in] args`: Array of JavaScript values as `napi_value` representing the
arguments of the function.
- `[in] context`: Context for the async operation that is invoking the callback.
This should normally be a value previously obtained from [Napi::AsyncContext](async_context.md).
However `nullptr` is also allowed, which indicates the current async context
(if any) is to be used for the callback.

Returns a `Napi::Value` representing the JavaScript value returned by the function.

## Operator

```cpp
Napi::Value Napi::Function::operator ()(const std::initializer_list<napi_value>& args) const;
```

- `[in] args`: Initializer list of JavaScript values as `napi_value`.

Returns a `Napi::Value` representing the JavaScript value returned by the function.
