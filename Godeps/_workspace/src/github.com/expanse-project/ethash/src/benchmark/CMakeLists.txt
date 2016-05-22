include_directories(..)

set(CMAKE_BUILD_TYPE Release)

if (MSVC)
  add_definitions("/openmp")
endif()

# enable C++11, should probably be a bit more specific about compiler
if (NOT MSVC)
  SET(CMAKE_CXX_FLAGS "-std=c++11")
endif()

if (NOT MPI_FOUND)
  find_package(MPI)
endif()

if (NOT CRYPTOPP_FOUND)
  find_package(CryptoPP 5.6.2)
endif()

if (CRYPTOPP_FOUND)
  add_definitions(-DWITH_CRYPTOPP)
  find_package (Threads REQUIRED)
endif()

if (NOT OpenCL_FOUND)
  find_package(OpenCL)
endif()
if (OpenCL_FOUND)
  add_definitions(-DWITH_OPENCL)
  include_directories(${OpenCL_INCLUDE_DIRS})
  list(APPEND FILES ethash_cl_miner.cpp ethash_cl_miner.h)
endif()

if (MPI_FOUND)
  include_directories(${MPI_INCLUDE_PATH})
  add_executable (Benchmark_MPI_FULL benchmark.cpp)
  target_link_libraries (Benchmark_MPI_FULL ${ETHHASH_LIBS} ${MPI_LIBRARIES} ${CMAKE_THREAD_LIBS_INIT})
  SET_TARGET_PROPERTIES(Benchmark_MPI_FULL PROPERTIES COMPILE_FLAGS "${COMPILE_FLAGS} ${MPI_COMPILE_FLAGS} -DFULL -DMPI")

  add_executable (Benchmark_MPI_LIGHT benchmark.cpp)
  target_link_libraries (Benchmark_MPI_LIGHT ${ETHHASH_LIBS} ${MPI_LIBRARIES} ${CMAKE_THREAD_LIBS_INIT})
  SET_TARGET_PROPERTIES(Benchmark_MPI_LIGHT PROPERTIES COMPILE_FLAGS "${COMPILE_FLAGS} ${MPI_COMPILE_FLAGS} -DMPI")
endif()

add_executable (Benchmark_FULL benchmark.cpp)
target_link_libraries (Benchmark_FULL ${ETHHASH_LIBS} ${CMAKE_THREAD_LIBS_INIT})
SET_TARGET_PROPERTIES(Benchmark_FULL PROPERTIES COMPILE_FLAGS "${COMPILE_FLAGS} -DFULL")

add_executable (Benchmark_LIGHT benchmark.cpp)
target_link_libraries (Benchmark_LIGHT ${ETHHASH_LIBS} ${CMAKE_THREAD_LIBS_INIT})

if (OpenCL_FOUND)
  add_executable (Benchmark_CL benchmark.cpp)
  target_link_libraries (Benchmark_CL ${ETHHASH_LIBS} ethash-cl ${CMAKE_THREAD_LIBS_INIT})
  SET_TARGET_PROPERTIES(Benchmark_CL PROPERTIES COMPILE_FLAGS "${COMPILE_FLAGS} -DOPENCL")
endif()