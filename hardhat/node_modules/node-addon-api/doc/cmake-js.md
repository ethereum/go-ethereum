# CMake.js

[**CMake.js**](https://github.com/cmake-js/cmake-js) is a build tool that allow native addon developers to compile their
C or C++ code into executable form. It works like **[node-gyp](node-gyp.md)** but
instead of Google's [**gyp**](https://gyp.gsrc.io) tool it is based on the [**CMake**](https://cmake.org) build system.

## Quick Start

### Install CMake

CMake.js requires that CMake be installed. Installers for a variety of platforms can be found on the [CMake website](https://cmake.org).

### Install CMake.js

For developers, CMake.js is typically installed as a global package:

```bash
npm install -g cmake-js
cmake-js --help
```

> For *users* of your native addon, CMake.js should be configured as a dependency in your `package.json` as described in the [CMake.js documentation](https://github.com/cmake-js/cmake-js).

### CMakeLists.txt

Your project will require a `CMakeLists.txt` file. The [CMake.js README file](https://github.com/cmake-js/cmake-js#usage) shows what's necessary.

### NAPI_VERSION

When building N-API addons, it's crucial to specify the N-API version your code is designed to work with. With CMake.js, this information is specified in the `CMakeLists.txt` file:

```
add_definitions(-DNAPI_VERSION=3)
```

Since N-API is ABI-stable, your N-API addon will work, without recompilation, with the N-API version you specify in `NAPI_VERSION` and all subsequent N-API versions.

In the absence of a need for features available only in a specific N-API version, version 3 is a good choice as it is the version of N-API that was active when N-API left experimental status.

### NAPI_EXPERIMENTAL

The following line in the `CMakeLists.txt` file will enable N-API experimental features if your code requires them:

```
add_definitions(-DNAPI_EXPERIMENTAL)
```

### node-addon-api

If your N-API native add-on uses the optional [**node-addon-api**](https://github.com/nodejs/node-addon-api#node-addon-api-module) C++ wrapper, the `CMakeLists.txt` file requires additional configuration information as described on the [CMake.js README file](https://github.com/cmake-js/cmake-js#n-api-and-node-addon-api).

## Example

A working example of an N-API native addon built using CMake.js can be found on the [node-addon-examples repository](https://github.com/nodejs/node-addon-examples/tree/master/build_with_cmake#building-n-api-addons-using-cmakejs).

## **CMake** Reference

  - [Installation](https://github.com/cmake-js/cmake-js#installation)
  - [How to use](https://github.com/cmake-js/cmake-js#usage)
  - [Using N-API and node-addon-api](https://github.com/cmake-js/cmake-js#n-api-and-node-addon-api)
  - [Tutorials](https://github.com/cmake-js/cmake-js#tutorials)
  - [Use case in the works - ArrayFire.js](https://github.com/cmake-js/cmake-js#use-case-in-the-works---arrayfirejs)

Sometimes finding the right settings is not easy so to accomplish at most
complicated task please refer to:

- [CMake documentation](https://cmake.org/)
- [CMake.js wiki](https://github.com/cmake-js/cmake-js/wiki)
