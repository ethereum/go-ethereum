# CallbackScope

There are cases (for example, resolving promises) where it is necessary to have
the equivalent of the scope associated with a callback in place when making
certain N-API calls.

## Methods

### Constructor

Creates a new callback scope on the stack.

```cpp
Napi::CallbackScope::CallbackScope(napi_env env, napi_callback_scope scope);
```

- `[in] env`: The environment in which to create the `Napi::CallbackScope`.
- `[in] scope`: The pre-existing `napi_callback_scope` or `Napi::CallbackScope`.

### Constructor

Creates a new callback scope on the stack.

```cpp
Napi::CallbackScope::CallbackScope(napi_env env, napi_async_context context);
```

- `[in] env`: The environment in which to create the `Napi::CallbackScope`.
- `[in] async_context`: The pre-existing `napi_async_context` or `Napi::AsyncContext`.

### Destructor

Deletes the instance of `Napi::CallbackScope` object.

```cpp
virtual Napi::CallbackScope::~CallbackScope();
```

### Env

```cpp
Napi::Env Napi::CallbackScope::Env() const;
```

Returns the `Napi::Env` associated with the `Napi::CallbackScope`.

## Operator

```cpp
Napi::CallbackScope::operator napi_callback_scope() const;
```

Returns the N-API `napi_callback_scope` wrapped by the `Napi::CallbackScope`
object. This can be used to mix usage of the C N-API and node-addon-api.
