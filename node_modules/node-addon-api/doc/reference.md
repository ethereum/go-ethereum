# Reference (template)

Holds a counted reference to a [`Napi::Value`](value.md) object; initially a weak reference unless otherwise specified, may be changed to/from a strong reference by adjusting the refcount.

The referenced `Napi::Value` is not immediately destroyed when the reference count is zero; it is merely then eligible for garbage-collection if there are no other references to the `Napi::Value`.

`Napi::Reference` objects allocated in static space, such as a global static instance, must call the `SuppressDestruct` method to prevent its destructor, running at program shutdown time, from attempting to reset the reference when the environment is no longer valid.

The following classes inherit, either directly or indirectly, from `Napi::Reference`:

* [`Napi::ObjectWrap`](object_wrap.md)
* [`Napi::ObjectReference`](object_reference.md)
* [`Napi::FunctionReference`](function_reference.md)

## Methods

### Factory Method

```cpp
static Napi::Reference<T> Napi::Reference::New(const T& value, uint32_t initialRefcount = 0);
```

* `[in] value`: The value which is to be referenced.

* `[in] initialRefcount`: The initial reference count.

### Empty Constructor

```cpp
Napi::Reference::Reference();
```

Creates a new _empty_ `Napi::Reference` instance.

### Constructor

```cpp
Napi::Reference::Reference(napi_env env, napi_value value);
```

* `[in] env`: The `napi_env` environment in which to construct the `Napi::Reference` object.

* `[in] value`: The N-API primitive value to be held by the `Napi::Reference`.

### Env

```cpp
Napi::Env Napi::Reference::Env() const;
```

Returns the `Napi::Env` value in which the `Napi::Reference` was instantiated.

### IsEmpty

```cpp
bool Napi::Reference::IsEmpty() const;
```

Determines whether the value held by the `Napi::Reference` is empty.

### Value

```cpp
T Napi::Reference::Value() const;
```

Returns the value held by the `Napi::Reference`.

### Ref

```cpp
uint32_t Napi::Reference::Ref();
```

Increments the reference count for the `Napi::Reference` and returns the resulting reference count. Throws an error if the increment fails.

### Unref

```cpp
uint32_t Napi::Reference::Unref();
```

Decrements the reference count for the `Napi::Reference` and returns the resulting reference count. Throws an error if the decrement fails.

### Reset (Empty)

```cpp
void Napi::Reference::Reset();
```

Sets the value held by the `Napi::Reference` to be empty.

### Reset

```cpp
void Napi::Reference::Reset(const T& value, uint32_t refcount = 0);
```

* `[in] value`: The value which is to be referenced.

* `[in] initialRefcount`: The initial reference count.

Sets the value held by the `Napi::Reference`.

### SuppressDestruct

```cpp
void Napi::Reference::SuppressDestruct();
```

Call this method on a `Napi::Reference` that is declared as static data to prevent its destructor, running at program shutdown time, from attempting to reset the reference when the environment is no longer valid.
