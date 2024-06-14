# Class property and descriptor

Property descriptor for use with `Napi::ObjectWrap::DefineClass()`.
This is different from the standalone `Napi::PropertyDescriptor` because it is
specific to each `Napi::ObjectWrap<T>` subclass.
This prevents using descriptors from a different class when defining a new class
(preventing the callbacks from having incorrect `this` pointers).

## Example

```cpp
#include <napi.h>

class Example : public Napi::ObjectWrap<Example> {
  public:
    static Napi::Object Init(Napi::Env env, Napi::Object exports);
    Example(const Napi::CallbackInfo &info);

  private:
    static Napi::FunctionReference constructor;
    double _value;
    Napi::Value GetValue(const Napi::CallbackInfo &info);
    void SetValue(const Napi::CallbackInfo &info, const Napi::Value &value);
};

Napi::Object Example::Init(Napi::Env env, Napi::Object exports) {
    Napi::Function func = DefineClass(env, "Example", {
        // Register a class instance accessor with getter and setter functions.
        InstanceAccessor("value", &Example::GetValue, &Example::SetValue),
        // We can also register a readonly accessor by passing nullptr as the setter.
        InstanceAccessor("readOnlyProp", &Example::GetValue, nullptr)
    });

    constructor = Napi::Persistent(func);
    constructor.SuppressDestruct();
    exports.Set("Example", func);

    return exports;
}

Example::Example(const Napi::CallbackInfo &info) : Napi::ObjectWrap<Example>(info) {
    Napi::Env env = info.Env();
    // ...
    Napi::Number value = info[0].As<Napi::Number>();
    this->_value = value.DoubleValue();
}

Napi::FunctionReference Example::constructor;

Napi::Value Example::GetValue(const Napi::CallbackInfo &info) {
    Napi::Env env = info.Env();
    return Napi::Number::New(env, this->_value);
}

void Example::SetValue(const Napi::CallbackInfo &info, const Napi::Value &value) {
    Napi::Env env = info.Env();
    // ...
    Napi::Number arg = value.As<Napi::Number>();
    this->_value = arg.DoubleValue();
}

// Initialize native add-on
Napi::Object Init (Napi::Env env, Napi::Object exports) {
    Example::Init(env, exports);
    return exports;
}

// Register and initialize native add-on
NODE_API_MODULE(NODE_GYP_MODULE_NAME, Init)
```

The above code can be used from JavaScript as follows:

```js
'use strict';

const { Example } = require('bindings')('addon');

const example = new Example(11);
console.log(example.value);
// It prints 11
example.value = 19;
console.log(example.value);
// It prints 19
example.readOnlyProp = 500;
console.log(example.readOnlyProp);
// Unchanged. It prints 19
```

## Methods

### Constructor

Creates new instance of `Napi::ClassPropertyDescriptor` descriptor object.

```cpp
Napi::ClassPropertyDescriptor(napi_property_descriptor desc) : _desc(desc) {}
```

- `[in] desc`: The `napi_property_descriptor`

Returns new instance of `Napi::ClassPropertyDescriptor` that is used as property descriptor
inside the `Napi::ObjectWrap<T>` class.

### Operator

```cpp
operator napi_property_descriptor&() { return _desc; }
```

Returns the original N-API `napi_property_descriptor` wrapped inside the `Napi::ClassPropertyDescriptor`

```cpp
operator const napi_property_descriptor&() const { return _desc; }
```

Returns the original N-API `napi_property_descriptor` wrapped inside the `Napi::ClassPropertyDescriptor`
