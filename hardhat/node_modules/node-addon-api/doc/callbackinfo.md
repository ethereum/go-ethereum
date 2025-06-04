# CallbackInfo

The object representing the components of the JavaScript request being made.

The `Napi::CallbackInfo` object is usually created and passed by the Node.js runtime or node-addon-api infrastructure.

The `Napi::CallbackInfo` object contains the arguments passed by the caller. The number of arguments is returned by the `Length` method. Each individual argument can be accessed using the `operator[]` method.

The `SetData` and `Data` methods are used to set and retrieve the data pointer contained in the `Napi::CallbackInfo` object.

## Methods

### Constructor

```cpp
Napi::CallbackInfo::CallbackInfo(napi_env env, napi_callback_info info);
```

- `[in] env`: The `napi_env` environment in which to construct the `Napi::CallbackInfo` object.
- `[in] info`: The `napi_callback_info` data structure from which to construct the `Napi::CallbackInfo` object.

### Env

```cpp
Napi::Env Napi::CallbackInfo::Env() const;
```

Returns the `Env` object in which the request is being made.

### NewTarget

```cpp
Napi::Value Napi::CallbackInfo::NewTarget() const;
```

Returns the `new.target` value of the constructor call. If the function that was invoked (and for which the `Napi::NCallbackInfo` was passed) is not a constructor call, a call to `IsEmpty()` on the returned value returns true.

### IsConstructCall

```cpp
bool Napi::CallbackInfo::IsConstructCall() const;
```

Returns a `bool` indicating if the function that was invoked (and for which the `Napi::CallbackInfo` was passed) is a constructor call.

### Length

```cpp
size_t Napi::CallbackInfo::Length() const;
```

Returns the number of arguments passed in the `Napi::CallbackInfo` object.

### operator []

```cpp
const Napi::Value operator [](size_t index) const;
```

- `[in] index`: The zero-based index of the requested argument.

Returns a `Napi::Value` object containing the requested argument.

### This

```cpp
Napi::Value Napi::CallbackInfo::This() const;
```

Returns the JavaScript `this` value for the call

### Data

```cpp
void* Napi::CallbackInfo::Data() const;
```

Returns the data pointer for the callback.

### SetData

```cpp
void Napi::CallbackInfo::SetData(void* data);
```

- `[in] data`: The new data pointer to associate with this `Napi::CallbackInfo` object.

Returns `void`.

### Not documented here

```cpp
Napi::CallbackInfo::~CallbackInfo();
// Disallow copying to prevent multiple free of _dynamicArgs
Napi::CallbackInfo::CallbackInfo(CallbackInfo const &) = delete;
void Napi::CallbackInfo::operator=(CallbackInfo const &) = delete;
```
