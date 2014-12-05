
cmake_minimum_required(VERSION 2.8.12)

project(evmjit)

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

# LLVM
find_package(LLVM REQUIRED CONFIG)
message(STATUS "Found LLVM ${LLVM_PACKAGE_VERSION}")
message(STATUS "Using LLVMConfig.cmake in: ${LLVM_DIR}")

add_subdirectory(libevmjit)
add_subdirectory(libevmjit-cpp)
add_subdirectory(evmcc)