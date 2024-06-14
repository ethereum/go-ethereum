# HandleScope

The HandleScope class is used to manage the lifetime of object handles
which are created through the use of node-addon-api. These handles
keep an object alive in the heap in order to ensure that the objects
are not collected while native code is using them.
A handle may be created when any new node-addon-api Value or one
of its subclasses is created or returned. For more details refer to
the section titled [Object lifetime management](object_lifetime_management.md).

## Methods

### Constructor

Creates a new handle scope on the stack.

```cpp
Napi::HandleScope::HandleScope(Napi::Env env);
```

- `[in] env`: The environment in which to construct the `Napi::HandleScope` object.

Returns a new `Napi::HandleScope`

### Constructor

Creates a new handle scope on the stack.

```cpp
Napi::HandleScope::HandleScope(Napi::Env env, Napi::HandleScope scope);
```

- `[in] env`: `Napi::Env` in which the scope passed in was created.
- `[in] scope`: pre-existing `Napi::HandleScope`.

Returns a new `Napi::HandleScope` instance which wraps the napi_handle_scope
handle passed in.  This can be used to mix usage of the C N-API
and node-addon-api.

operator HandleScope::napi_handle_scope

```cpp
operator Napi::HandleScope::napi_handle_scope() const
```

Returns the N-API napi_handle_scope wrapped by the `Napi::EscapableHandleScope` object.
This can be used to mix usage of the C N-API and node-addon-api by allowing
the class to be used be converted to a napi_handle_scope.

### Destructor
```cpp
Napi::HandleScope::~HandleScope();
```

Deletes the `Napi::HandleScope` instance and allows any objects/handles created
in the scope to be collected by the garbage collector.  There is no
guarantee as to when the gargbage collector will do this.

### Env

```cpp
Napi::Env Napi::HandleScope::Env() const;
```

Returns the `Napi::Env` associated with the `Napi::HandleScope`.
