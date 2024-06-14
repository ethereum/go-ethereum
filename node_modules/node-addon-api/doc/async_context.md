# AsyncContext

The [Napi::AsyncWorker](async_worker.md) class may not be appropriate for every
scenario. When using any other async mechanism, introducing a new class
`Napi::AsyncContext` is necessary to ensure an async operation is properly
tracked by the runtime. The `Napi::AsyncContext` class can be passed to
[Napi::Function::MakeCallback()](function.md) method to properly restore the
correct async execution context.

## Methods

### Constructor

Creates a new `Napi::AsyncContext`.

```cpp
explicit Napi::AsyncContext::AsyncContext(napi_env env, const char* resource_name);
```

- `[in] env`: The environment in which to create the `Napi::AsyncContext`.
- `[in] resource_name`: Null-terminated strings that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the `async_hooks` API.

### Constructor

Creates a new `Napi::AsyncContext`.

```cpp
explicit Napi::AsyncContext::AsyncContext(napi_env env, const char* resource_name, const Napi::Object& resource);
```

- `[in] env`: The environment in which to create the `Napi::AsyncContext`.
- `[in] resource_name`: Null-terminated strings that represents the
identifier for the kind of resource that is being provided for diagnostic
information exposed by the `async_hooks` API.
- `[in] resource`: Object associated with the asynchronous operation that
will be passed to possible `async_hooks`.

### Destructor

The `Napi::AsyncContext` to be destroyed.

```cpp
virtual Napi::AsyncContext::~AsyncContext();
```

### Env

Requests the environment in which the async context has been initially created.

```cpp
Napi::Env Env() const;
```

Returns the `Napi::Env` environment in which the async context has been created.

## Operator

```cpp
Napi::AsyncContext::operator napi_async_context() const;
```

Returns the N-API `napi_async_context` wrapped by the `Napi::AsyncContext`
object. This can be used to mix usage of the C N-API and node-addon-api.

## Example

```cpp
#include "napi.h"

void MakeCallbackWithAsyncContext(const Napi::CallbackInfo& info) {
  Napi::Function callback = info[0].As<Napi::Function>();
  Napi::Object resource = info[1].As<Napi::Object>();

  // Creat a new async context instance.
  Napi::AsyncContext context(info.Env(), "async_context_test", resource);

  // Invoke the callback with the async context instance.
  callback.MakeCallback(Napi::Object::New(info.Env()),
      std::initializer_list<napi_value>{}, context);

  // The async context instance is automatically destroyed here because it's
  // block-scope like `Napi::HandleScope`.
}
```
