# MemoryManagement

The `Napi::MemoryManagement` class contains functions that give the JavaScript engine
an indication of the amount of externally allocated memory that is kept alive by
JavaScript objects.

## Methods

### AdjustExternalMemory

The function `AdjustExternalMemory` adjusts the amount of registered external
memory used to give the JavaScript engine an indication of the amount of externally
allocated memory that is kept alive by JavaScript objects.
The JavaScript engine uses this to decide when to perform global garbage collections.
Registering externally allocated memory will trigger global garbage collections
more often than it would otherwise in an attempt to garbage collect the JavaScript
objects that keep the externally allocated memory alive.

```cpp
static int64_t Napi::MemoryManagement::AdjustExternalMemory(Napi::Env env, int64_t change_in_bytes);
```

- `[in] env`: The environment in which the API is invoked under.
- `[in] change_in_bytes`: The change in externally allocated memory that is kept
alive by JavaScript objects expressed in bytes.

Returns the adjusted memory value.
