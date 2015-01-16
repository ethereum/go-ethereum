cmake_minimum_required(VERSION 2.8.12)

project(evmjit)

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

if(CMAKE_CXX_COMPILER_ID STREQUAL "MSVC")
else()
	set(CMAKE_CXX_FLAGS "-std=c++11 -Wall -Wno-unknown-pragmas -Wextra -fPIC")
endif()

if(CMAKE_SYSTEM_NAME STREQUAL "Linux")
	# Do not allow unresovled symbols in shared library (default on linux)
	set(CMAKE_SHARED_LINKER_FLAGS "-Wl,--no-undefined")
endif()

# LLVM
if(LLVM_DIR OR APPLE)	# local LLVM build
	find_package(LLVM REQUIRED CONFIG)
	message(STATUS "Found LLVM ${LLVM_PACKAGE_VERSION}")
	message(STATUS "Using LLVMConfig.cmake in: ${LLVM_DIR}")
	add_definitions(${LLVM_DEFINITIONS})
	# TODO: bitwriter is needed only for evmcc
	llvm_map_components_to_libnames(LLVM_LIBS core support mcjit x86asmparser x86codegen bitwriter)
else()
	# Workaround for Ubuntu broken LLVM package
	message(STATUS "Using llvm-3.5-dev package from Ubuntu. If does not work, build LLVM and set -DLLVM_DIR=llvm-build/share/llvm/cmake")
	execute_process(COMMAND llvm-config-3.5 --includedir OUTPUT_VARIABLE LLVM_INCLUDE_DIRS)
	message(STATUS "LLVM include dirs: ${LLVM_INCLUDE_DIRS}")
	set(LLVM_LIBS "-lLLVMBitWriter -lLLVMX86CodeGen -lLLVMSelectionDAG -lLLVMAsmPrinter -lLLVMCodeGen -lLLVMScalarOpts -lLLVMInstCombine -lLLVMTransformUtils -lLLVMipa -lLLVMAnalysis -lLLVMX86AsmParser -lLLVMX86Desc -lLLVMX86Info -lLLVMX86AsmPrinter -lLLVMX86Utils -lLLVMMCJIT -lLLVMTarget -lLLVMRuntimeDyld -lLLVMObject -lLLVMMCParser -lLLVMBitReader -lLLVMExecutionEngine -lLLVMMC -lLLVMCore -lLLVMSupport -lz -lpthread -lffi -ltinfo -ldl -lm")
	add_definitions(-D__STDC_CONSTANT_MACROS -D__STDC_FORMAT_MACROS -D__STDC_LIMIT_MACROS)
	link_directories(/usr/lib/llvm-3.5/lib)
endif()

# Boost
find_package(Boost REQUIRED)

add_subdirectory(libevmjit)

if(EVMJIT_CPP)
	add_subdirectory(libevmjit-cpp)
endif()

if(EVMJIT_TOOLS)
	add_subdirectory(evmcc)
endif()
