# EscapableHandleScope

The `Napi::EscapableHandleScope` class is used to manage the lifetime of object handles
which are created through the use of node-addon-api. These handles
keep an object alive in the heap in order to ensure that the objects
are not collected by the garbage collector while native code is using them.
A handle may be created when any new node-addon-api Value or one
of its subclasses is created or returned.

The `Napi::EscapableHandleScope` is a special type of `Napi::HandleScope`
which allows a single handle to be "promoted" to an outer scope.

For more details refer to the section titled
[Object lifetime management](object_lifetime_management.md).

## Methods

### Constructor

Creates a new escapable handle scope.

```cpp
Napi::EscapableHandleScope Napi::EscapableHandleScope::New(Napi:Env env);
```

- `[in] Env`: The environment in which to construct the `Napi::EscapableHandleScope` object.

Returns a new `Napi::EscapableHandleScope`

### Constructor

Creates a new escapable handle scope.

```cpp
Napi::EscapableHandleScope Napi::EscapableHandleScope::New(napi_env env, napi_handle_scope scope);
```

- `[in] env`: napi_env in which the scope passed in was created.
- `[in] scope`: pre-existing napi_handle_scope.

Returns a new `Napi::EscapableHandleScope` instance which wraps the
napi_escapable_handle_scope handle passed in. This can be used
to mix usage of the C N-API and node-addon-api.

operator EscapableHandleScope::napi_escapable_handle_scope

```cpp
operator Napi::EscapableHandleScope::napi_escapable_handle_scope() const
```

Returns the N-API napi_escapable_handle_scope wrapped by the `Napi::EscapableHandleScope` object.
This can be used to mix usage of the C N-API and node-addon-api by allowing
the class to be used be converted to a napi_escapable_handle_scope.

### Destructor
```cpp
Napi::EscapableHandleScope::~EscapableHandleScope();
```

Deletes the `Napi::EscapableHandleScope` instance and allows any objects/handles created
in the scope to be collected by the garbage collector. There is no
guarantee as to when the gargbage collector will do this.

### Escape

```cpp
napi::Value Napi::EscapableHandleScope::Escape(napi_value escapee);
```

- `[in] escapee`: Napi::Value or napi_env to promote to the outer scope

Returns `Napi::Value` which can be used in the outer scope. This method can
be called at most once on a given `Napi::EscapableHandleScope`. If it is called
more than once an exception will be thrown.

### Env

```cpp
Napi::Env Napi::EscapableHandleScope::Env() const;
```

Returns the `Napi::Env` associated with the `Napi::EscapableHandleScope`.
