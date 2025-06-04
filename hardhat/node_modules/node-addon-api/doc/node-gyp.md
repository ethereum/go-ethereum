# node-gyp

C++ code needs to be compiled into executable form whether it be as an object
file to linked with others, a shared library, or a standalone executable.

The main reason for this is that we need to link to the Node.js dependencies and
headers correctly, another reason is that we need a cross platform way to build
C++ source into binary for the target platform.

Until now **node-gyp** is the **de-facto** standard build tool for writing
Node.js addons. It's based on Google's **gyp** build tool, which abstract away
many of the tedious issues related to cross platform building.

**node-gyp** uses a file called ```binding.gyp``` that is located on the root of
your addon project.

```binding.gyp``` file, contains all building configurations organized with a
JSON like syntax. The most important parameter is the  **target** that must be
set to the same value used on the initialization code of the addon as in the
examples reported below:

### **binding.gyp**

```gyp
{
  "targets": [
    {
      # myModule is the name of your native addon
      "target_name": "myModule",
      "sources": ["src/my_module.cc", ...],
      ...
  ]
}
```

### **my_module.cc**

```cpp
#include <napi.h>

// ...

/**
* This code is our entry-point. We receive two arguments here, the first is the
* environment that represent an independent instance of the JavaScript runtime,
* the second is exports, the same as module.exports in a .js file.
* You can either add properties to the exports object passed in or create your
* own exports object. In either case you must return the object to be used as
* the exports for the module when you return from the Init function.
*/
Napi::Object Init(Napi::Env env, Napi::Object exports) {

  // ...

  return exports;
}

/**
* This code defines the entry-point for the Node addon, it tells Node where to go
* once the library has been loaded into active memory. The first argument must
* match the "target" in our *binding.gyp*. Using NODE_GYP_MODULE_NAME ensures
* that the argument will be correct, as long as the module is built with
* node-gyp (which is the usual way of building modules). The second argument
* points to the function to invoke. The function must not be namespaced.
*/
NODE_API_MODULE(NODE_GYP_MODULE_NAME, Init)
```

## **node-gyp** reference

  - [Installation](https://www.npmjs.com/package/node-gyp#installation)
  - [How to use](https://www.npmjs.com/package/node-gyp#how-to-use)
  - [The binding.gyp file](https://www.npmjs.com/package/node-gyp#the-bindinggyp-file)
  - [Commands](https://www.npmjs.com/package/node-gyp#commands)
  - [Command options](https://www.npmjs.com/package/node-gyp#command-options)
  - [Configuration](https://www.npmjs.com/package/node-gyp#configuration)

Sometimes finding the right settings for ```binding.gyp``` is not easy so to
accomplish at most complicated task please refer to:

- [GYP documentation](https://gyp.gsrc.io/index.md)
- [node-gyp wiki](https://github.com/nodejs/node-gyp/wiki)
