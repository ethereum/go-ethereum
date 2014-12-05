cmake_policy(SET CMP0015 NEW)

project(evmjit CXX)

file(GLOB SOURCES "*.cpp")
file(GLOB HEADERS "*.h")
set_property(TARGET ${PROJECT_NAME} PROPERTY FOLDER "tools")

set(EXECUTABLE evmcc)
add_executable(${EXECUTABLE} ${HEADERS} ${SOURCES})

include_directories(../..)

target_link_libraries(${EXECUTABLE} boost_program_options)
target_link_libraries(${EXECUTABLE} devcore)
target_link_libraries(${EXECUTABLE} ethcore)
target_link_libraries(${EXECUTABLE} evmjit)
target_link_libraries(${EXECUTABLE} evm)
target_link_libraries(${EXECUTABLE} ethereum)

if ("${TARGET_PLATFORM}" STREQUAL "w64")
	set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -static-libgcc -static-libstdc++")
	target_link_libraries(${EXECUTABLE} gcc)
	target_link_libraries(${EXECUTABLE} gdi32)
	target_link_libraries(${EXECUTABLE} ws2_32)
	target_link_libraries(${EXECUTABLE} mswsock)
	target_link_libraries(${EXECUTABLE} shlwapi)
	target_link_libraries(${EXECUTABLE} iphlpapi)
	target_link_libraries(${EXECUTABLE} boost_thread_win32-mt-s)
	set(CMAKE_SHARED_LIBRARY_LINK_CXX_FLAGS)
else ()
	find_package(Threads REQUIRED)
	target_link_libraries(${EXECUTABLE} ${CMAKE_THREAD_LIBS_INIT})
endif ()

# LLVM specific commands
find_package(LLVM REQUIRED CONFIG)
include_directories(${LLVM_INCLUDE_DIRS})
add_definitions(${LLVM_DEFINITIONS})

llvm_map_components_to_libnames(llvm_libs bitwriter)
target_link_libraries(evmcc ${llvm_libs})
# end of LLVM specific commands


install( TARGETS ${EXECUTABLE} DESTINATION bin )

cmake_policy(SET CMP0015 NEW)
