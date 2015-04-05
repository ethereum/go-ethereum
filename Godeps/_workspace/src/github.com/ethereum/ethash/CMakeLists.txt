cmake_minimum_required(VERSION 2.8.7)
project(ethash)

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_SOURCE_DIR}/cmake/modules/")
set(ETHHASH_LIBS ethash)

if (WIN32 AND WANT_CRYPTOPP)
	add_subdirectory(cryptopp)
endif()

add_subdirectory(src/libethash)
# bin2h.cmake doesn't work
if (NOT OpenCL_FOUND)
	find_package(OpenCL)
endif()

if (OpenCL_FOUND)
	add_subdirectory(src/libethash-cl)
endif()
add_subdirectory(src/benchmark EXCLUDE_FROM_ALL)
add_subdirectory(test/c)
