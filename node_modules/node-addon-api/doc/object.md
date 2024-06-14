# Object

The `Napi::Object` class corresponds to a JavaScript object. It is extended by the following node-addon-api classes that you may use when working with more specific types:

- [`Napi::Value`](value.md) and extends [`Napi::Array`](array.md)
- [`Napi::ArrayBuffer`](array_buffer.md)
- [`Napi::Buffer<T>`](buffer.md)
- [`Napi::Function`](function.md)
- [`Napi::TypedArray`](typed_array.md).

This class provides a number of convenience methods, most of which are used to set or get properties on a JavaScript object. For example, Set() and Get().

## Example
```cpp
#include <napi.h>

using namespace Napi;

Void Init(Env env) {

  // Create a new instance
  Object obj = Object::New(env);

  // Assign values to properties
  obj.Set("hello", "world");
  obj.Set(uint32_t(42), "The Answer to Life, the Universe, and Everything");
  obj.Set("Douglas Adams", true);

  // Get properties
  Value val1 = obj.Get("hello");
  Value val2 = obj.Get(uint32_t(42));
  Value val3 = obj.Get("Douglas Adams");

  // Test if objects have properties.
  bool obj1 = obj.Has("hello"); // true
  bool obj2 = obj.Has("world"); // false

}
```

## Methods

### Empty Constructor

```cpp
Napi::Object::Object();
```
Creates a new empty Object instance.

### Constructor

```cpp
Napi::Object::Object(napi_env env, napi_value value);
```
- `[in] env`: The `napi_env` environment in which to construct the Value object.

- `[in] value`: The C++ primitive from which to instantiate the Value. `value` may be any of:
  - bool
  - Any integer type
  - Any floating point type
  - const char* (encoded using UTF-8, null-terminated)
  - const char16_t* (encoded using UTF-16-LE, null-terminated)
  - std::string (encoded using UTF-8)
  - std::u16string
  - Napi::Value
  - napi_value

Creates a non-empty `Napi::Object` instance.

### New()

```cpp
Napi::Object Napi::Object::New(napi_env env);
```
- `[in] env`: The `napi_env` environment in which to construct the `Napi::Value` object.

Creates a new `Napi::Object` value.

### Set()

```cpp
void Napi::Object::Set (____ key, ____ value);
```
- `[in] key`: The name for the property being assigned.
- `[in] value`: The value being assigned to the property.

Add a property with the specified key with the specified value to the object.

The key can be any of the following types:
- `napi_value`
- [`Napi::Value`](value.md)
- `const char*`
- `const std::string&`
- `uint32_t`

While the value must be any of the following types:
- `napi_value`
- [`Napi::Value`](value.md)
- `const char*`
- `std::string&`
- `bool`
- `double`

### Get()

```cpp
Napi::Value Napi::Object::Get(____ key);
```
- `[in] key`: The name of the property to return the value for.

Returns the [`Napi::Value`](value.md) associated with the key property. Returns the value *undefined* if the key does not exist.

The `key` can be any of the following types:
- `napi_value`
- [`Napi::Value`](value.md)
- `const char *`
- `const std::string &`
- `uint32_t`

### Has()

```cpp
bool Napi::Object::Has (____ key) const;
```
- `[in] key`: The name of the property to check.

Returns a `bool` that is *true* if the object has a property named `key` and *false* otherwise.

### InstanceOf()

```cpp
bool Napi::Object::InstanceOf (const Function& constructor) const
```
- `[in] constructor`: The constructor [`Napi::Function`](function.md) of the value that is being compared with the object.

Returns a `bool` that is true if the `Napi::Object` is an instance created by the `constructor` and false otherwise.

Note: This is equivalent to the JavaScript instanceof operator.

### AddFinalizer()
```cpp
template <typename Finalizer, typename T>
inline void AddFinalizer(Finalizer finalizeCallback, T* data);
```

- `[in] finalizeCallback`: The function to call when the object is garbage-collected.
- `[in] data`: The data to associate with the object.

Associates `data` with the object, calling `finalizeCallback` when the object is garbage-collected. `finalizeCallback`
has the signature
```cpp
void finalizeCallback(Napi::Env env, T* data);
```
where `data` is the pointer that was passed into the call to `AddFinalizer()`.

### AddFinalizer()
```cpp
template <typename Finalizer, typename T, typename Hint>
inline void AddFinalizer(Finalizer finalizeCallback,
                         T* data,
                         Hint* finalizeHint);
```

- `[in] data`: The data to associate with the object.
- `[in] finalizeCallback`: The function to call when the object is garbage-collected.

Associates `data` with the object, calling `finalizeCallback` when the object is garbage-collected. An additional hint
may be given. It will also be passed to `finalizeCallback`, which has the signature
```cpp
void finalizeCallback(Napi::Env env, T* data, Hint* hint);
```
where `data` and `hint` are the pointers that were passed into the call to `AddFinalizer()`.

### DefineProperty()

```cpp
void Napi::Object::DefineProperty (const Napi::PropertyDescriptor& property);
```
- `[in] property`: A [`Napi::PropertyDescriptor`](property_descriptor.md).

Define a property on the object.

### DefineProperties()

```cpp
void Napi::Object::DefineProperties (____ properties)
```
- `[in] properties`: A list of [`Napi::PropertyDescriptor`](property_descriptor.md). Can be one of the following types:
	- const std::initializer_list<Napi::PropertyDescriptor>&
	- const std::vector<Napi::PropertyDescriptor>&

Defines properties on the object.

### Operator[]()

```cpp
Napi::PropertyLValue<std::string> Napi::Object::operator[] (const char* utf8name);
```
- `[in] utf8name`: UTF-8 encoded null-terminated property name.

Returns a [`Napi::PropertyLValue`](propertylvalue.md) as the named property or sets the named property.

```cpp
Napi::PropertyLValue<std::string> Napi::Object::operator[] (const std::string& utf8name);
```
- `[in] utf8name`: UTF-8 encoded property name.

Returns a [`Napi::PropertyLValue`](propertylvalue.md) as the named property or sets the named property.

```cpp
Napi::PropertyLValue<uint32_t> Napi::Object::operator[] (uint32_t index);
```
- `[in] index`: Element index.

Returns a [`Napi::PropertyLValue`](propertylvalue.md) or sets an indexed property or array element.

```cpp
Napi::Value Napi::Object::operator[] (const char* utf8name) const;
```
- `[in] utf8name`: UTF-8 encoded null-terminated property name.

Returns the named property as a [`Napi::Value`](value.md).

```cpp
Napi::Value Napi::Object::operator[] (const std::string& utf8name) const;
```
- `[in] utf8name`: UTF-8 encoded property name.

Returns the named property as a [`Napi::Value`](value.md).

```cpp
Napi::Value Napi::Object::operator[] (uint32_t index) const;
```
- `[in] index`: Element index.

Returns an indexed property or array element as a [`Napi::Value`](value.md).
