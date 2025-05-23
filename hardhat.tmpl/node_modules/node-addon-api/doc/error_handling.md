# Error handling

Error handling represents one of the most important considerations when
implementing a Node.js native add-on. When an error occurs in your C++ code you
have to handle and dispatch it correctly. **node-addon-api** uses return values and
JavaScript exceptions for error handling. You can choose return values or
exception handling based on the mechanism that works best for your add-on.

The `Napi::Error` is a persistent reference (for more info see: [`Napi::ObjectReference`](object_reference.md))
to a JavaScript error object. Use of this class depends on whether C++
exceptions are enabled at compile time.

If C++ exceptions are enabled (for more info see: [Setup](setup.md)), then the
`Napi::Error` class extends `std::exception` and enables integrated
error-handling for C++ exceptions and JavaScript exceptions.

The following sections explain the approach for each case:

- [Handling Errors With C++ Exceptions](#exceptions)
- [Handling Errors Without C++ Exceptions](#noexceptions)

<a name="exceptions"></a>

In most cases when an error occurs, the addon should do whatever clean is possible
and then return to JavaScript so that they error can be propagated.  In less frequent
cases the addon may be able to recover from the error, clear the error and then
continue.

## Handling Errors With C++ Exceptions

When C++ exceptions are enabled try/catch can be used to catch exceptions thrown
from calls to JavaScript and then they can either be handled or rethrown before
returning from a native method.

If a node-addon-api call fails without executing any JavaScript code (for example due to
an invalid argument), then node-addon-api automatically converts and throws
the error as a C++ exception of type `Napi::Error`.

If a JavaScript function called by C++ code via node-addon-api throws a JavaScript
exception, then node-addon-api automatically converts and throws it as a C++
exception of type `Napi:Error` on return from the JavaScript code to the native
method.

If a C++ exception of type `Napi::Error` escapes from a N-API C++ callback, then
the N-API wrapper automatically converts and throws it as a JavaScript exception.

On return from a native method, node-addon-api will automatically convert a pending C++
exception to a JavaScript exception.

When C++ exceptions are enabled try/catch can be used to catch exceptions thrown
from calls to JavaScript and then they can either be handled or rethrown before
returning from a native method.

## Examples with C++ exceptions enabled

### Throwing a C++ exception

```cpp
Env env = ...
throw Napi::Error::New(env, "Example exception");
// other C++ statements
// ...
```

The statements following the throw statement will not be executed. The exception
will bubble up as a C++ exception of type `Napi::Error`, until it is either caught
while still in C++, or else automatically propagated as a JavaScript exception
when returning to JavaScript.

### Propagating a N-API C++ exception

```cpp
Napi::Function jsFunctionThatThrows = someObj.As<Napi::Function>();
Napi::Value result = jsFunctionThatThrows({ arg1, arg2 });
// other C++ statements
// ...
```

The C++ statements following the call to the JavaScript function will not be
executed. The exception will bubble up as a C++ exception of type `Napi::Error`,
until it is either caught while still in C++, or else automatically propagated as
a JavaScript exception when returning to JavaScript.

### Handling a N-API C++ exception

```cpp
Napi::Function jsFunctionThatThrows = someObj.As<Napi::Function>();
Napi::Value result;
try {
    result = jsFunctionThatThrows({ arg1, arg2 });
} catch (const Error& e) {
    cerr << "Caught JavaScript exception: " + e.what();
}
```

Since the exception was caught here, it will not be propagated as a JavaScript
exception.

<a name="noexceptions"></a>

## Handling Errors Without C++ Exceptions

If C++ exceptions are disabled (for more info see: [Setup](setup.md)), then the
`Napi::Error` class does not extend `std::exception`. This means that any calls to
node-addon-api function do not throw a C++ exceptions. Instead, it raises
_pending_ JavaScript exceptions and returns an _empty_ `Napi::Value`.
The calling code should check `env.IsExceptionPending()` before attempting to use a
returned value, and may use methods on the `Napi::Env` class
to check for, get, and clear a pending JavaScript exception (for more info see: [Env](env.md)).
If the pending exception is not cleared, it will be thrown when the native code
returns to JavaScript.

## Examples with C++ exceptions disabled

### Throwing a JS exception

```cpp
Napi::Env env = ...
Napi::Error::New(env, "Example exception").ThrowAsJavaScriptException();
return;
```

After throwing a JavaScript exception, the code should generally return
immediately from the native callback, after performing any necessary cleanup.

### Propagating a N-API JS exception

```cpp
Napi::Env env = ...
Napi::Function jsFunctionThatThrows = someObj.As<Napi::Function>();
Napi::Value result = jsFunctionThatThrows({ arg1, arg2 });
if (env.IsExceptionPending()) {
    Error e = env.GetAndClearPendingException();
    return e.Value();
}
```

If env.IsExceptionPending() returns true a JavaScript exception is pending. To
let the exception propagate, the code should generally return immediately from
the native callback, after performing any necessary cleanup.

### Handling a N-API JS exception

```cpp
Napi::Env env = ...
Napi::Function jsFunctionThatThrows = someObj.As<Napi::Function>();
Napi::Value result = jsFunctionThatThrows({ arg1, arg2 });
if (env.IsExceptionPending()) {
    Napi::Error e = env.GetAndClearPendingException();
    cerr << "Caught JavaScript exception: " + e.Message();
}
```

Since the exception was cleared here, it will not be propagated as a JavaScript
exception after the native callback returns.

## Calling N-API directly from a **node-addon-api** addon

**node-addon-api** provides macros for throwing errors in response to non-OK
`napi_status` results when calling [N-API](https://nodejs.org/docs/latest/api/n-api.html)
functions from within a native addon. These macros are defined differently
depending on whether C++ exceptions are enabled or not, but are available for
use in either case.

### `NAPI_THROW(e, ...)`

This macro accepts a `Napi::Error`, throws it, and returns the value given as
the last parameter. If C++ exceptions are enabled (by defining
`NAPI_CPP_EXCEPTIONS` during the build), the return value will be ignored.

### `NAPI_THROW_IF_FAILED(env, status, ...)`

This macro accepts a `Napi::Env` and a `napi_status`. It constructs an error
from the `napi_status`, throws it, and returns the value given as the last
parameter. If C++ exceptions are enabled (by defining `NAPI_CPP_EXCEPTIONS`
during the build), the return value will be ignored.

### `NAPI_THROW_IF_FAILED_VOID(env, status)`

This macro accepts a `Napi::Env` and a `napi_status`. It constructs an error
from the `napi_status`, throws it, and returns.

### `NAPI_FATAL_IF_FAILED(status, location, message)`

This macro accepts a `napi_status`, a C string indicating the location where the
error occurred, and a second C string for the message to display.
