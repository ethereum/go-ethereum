
cmake_minimum_required(VERSION 2.8.12)

project(evmjit)

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

add_subdirectory(libevmjit)
add_subdirectory(libevmjit-cpp)