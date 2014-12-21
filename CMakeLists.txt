cmake_minimum_required(VERSION 2.8.12)

project(evmjit)

set_property(GLOBAL PROPERTY USE_FOLDERS ON)

# LLVM
if(LLVM_DIR)	# local LLVM build
	find_package(LLVM REQUIRED CONFIG)
	message(STATUS "Found LLVM ${LLVM_PACKAGE_VERSION}")
	message(STATUS "Using LLVMConfig.cmake in: ${LLVM_DIR}")
	# TODO: bitwriter is needed only for evmcc
	llvm_map_components_to_libnames(LLVM_LIBS core support mcjit x86asmparser x86codegen bitwriter)
else()
	# Workaround for Ubuntu broken LLVM package
	message(STATUS "Using llvm-3.5-dev package from Ubuntu. If does not work, build LLVM and set -DLLVM_DIR=llvm-build/share/llvm/cmake")
	execute_process(COMMAND llvm-config-3.5 --includedir OUTPUT_VARIABLE LLVM_INCLUDE_DIRS)
	message(STATUS "LLVM include dirs: ${LLVM_INCLUDE_DIRS}")
	set(LLVM_LIBS "-lLLVMBitWriter -lLLVMX86CodeGen -lLLVMSelectionDAG -lLLVMAsmPrinter -lLLVMCodeGen -lLLVMScalarOpts -lLLVMInstCombine -lLLVMTransformUtils -lLLVMipa -lLLVMAnalysis -lLLVMX86AsmParser -lLLVMX86Desc -lLLVMX86Info -lLLVMX86AsmPrinter -lLLVMX86Utils -lLLVMMCJIT -lLLVMTarget -lLLVMRuntimeDyld -lLLVMObject -lLLVMMCParser -lLLVMBitReader -lLLVMExecutionEngine -lLLVMMC -lLLVMCore -lLLVMSupport -lz -lpthread -lffi -ltinfo -ldl -lm")
endif()

# Boost
find_package(Boost REQUIRED)

add_subdirectory(libevmjit)
add_subdirectory(libevmjit-cpp)
add_subdirectory(evmcc)