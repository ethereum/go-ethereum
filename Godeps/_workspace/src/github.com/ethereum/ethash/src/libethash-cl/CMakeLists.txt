cmake_minimum_required(VERSION 2.8)

set(LIBRARY ethash-cl)
set(CMAKE_BUILD_TYPE Release)

include(bin2h.cmake)
bin2h(SOURCE_FILE ethash_cl_miner_kernel.cl
      VARIABLE_NAME ethash_cl_miner_kernel
      HEADER_FILE ${CMAKE_CURRENT_BINARY_DIR}/ethash_cl_miner_kernel.h)

if (NOT MSVC)
	# Initialize CXXFLAGS for c++11
    set(CMAKE_CXX_FLAGS                "-Wall -std=c++11")
    set(CMAKE_CXX_FLAGS_DEBUG          "-O0 -g")
    set(CMAKE_CXX_FLAGS_MINSIZEREL     "-Os -DNDEBUG")
    set(CMAKE_CXX_FLAGS_RELEASE        "-O3 -DNDEBUG")
    set(CMAKE_CXX_FLAGS_RELWITHDEBINFO "-O2 -g")

    # Compiler-specific C++11 activation.
    if ("${CMAKE_CXX_COMPILER_ID}" MATCHES "GNU")
        execute_process(
            COMMAND ${CMAKE_CXX_COMPILER} -dumpversion OUTPUT_VARIABLE GCC_VERSION)
        if (NOT (GCC_VERSION VERSION_GREATER 4.7 OR GCC_VERSION VERSION_EQUAL 4.7))
            message(FATAL_ERROR "${PROJECT_NAME} requires g++ 4.7 or greater.")
        endif ()
    elseif ("${CMAKE_CXX_COMPILER_ID}" MATCHES "Clang")
        set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -stdlib=libc++")
    else ()
        message(FATAL_ERROR "Your C++ compiler does not support C++11.")
    endif ()
endif()

set(OpenCL_FOUND TRUE)
set(OpenCL_INCLUDE_DIRS /usr/include/CL)
set(OpenCL_LIBRARIES -lOpenCL)

if (NOT OpenCL_FOUND)
	find_package(OpenCL)
endif()

if (OpenCL_FOUND)
	set(CMAKE_CXX_FLAGS "-std=c++11 -Wall -Wno-unknown-pragmas -Wextra -Werror -pedantic -fPIC ${CMAKE_CXX_FLAGS}")
	include_directories(${OpenCL_INCLUDE_DIRS} ${CMAKE_CURRENT_BINARY_DIR})
	include_directories(..)
	add_library(${LIBRARY} ethash_cl_miner.cpp ethash_cl_miner.h cl.hpp)
	TARGET_LINK_LIBRARIES(${LIBRARY} ${OpenCL_LIBRARIES} ethash)
endif()
