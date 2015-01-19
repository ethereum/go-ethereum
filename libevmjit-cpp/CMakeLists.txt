set(TARGET_NAME evmjit-cpp)

# Boost
find_package(Boost REQUIRED)

set(SOURCES
    Env.cpp     
    JitVM.cpp   JitVM.h
)
source_group("" FILES ${SOURCES})

add_library(${TARGET_NAME} ${SOURCES})
set_property(TARGET ${TARGET_NAME} PROPERTY FOLDER "libs")

include_directories(../..)
include_directories(${LLVM_INCLUDE_DIRS})
include_directories(${Boost_INCLUDE_DIRS})

target_link_libraries(${TARGET_NAME} evmjit)
