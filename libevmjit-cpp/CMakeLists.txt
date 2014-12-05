project(evmjit-cpp CXX)

set(SOURCES
    Env.cpp     
    JitVM.cpp   JitVM.h
)

add_library(${PROJECT_NAME} ${SOURCES})

include_directories(../..)

find_package(LLVM REQUIRED CONFIG)
include_directories(${LLVM_INCLUDE_DIRS})
add_definitions(${LLVM_DEFINITIONS})

source_group("" FILES ${SOURCES})
set_property(TARGET ${PROJECT_NAME} PROPERTY FOLDER "libs")