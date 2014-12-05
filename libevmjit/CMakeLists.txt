project(evmjit CXX)

file(GLOB SOURCES "*.cpp")
file(GLOB HEADERS "*.h")

add_library(${PROJECT_NAME} ${SOURCES} ${HEADERS})

#include_directories(..)

#target_link_libraries(${EXECUTABLE} devcore)
#target_link_libraries(${EXECUTABLE} ethcore)
#target_link_libraries(${EXECUTABLE} evmcore)


# if ("${TARGET_PLATFORM}" STREQUAL "w64")
	# set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -static-libgcc -static-libstdc++")
	# target_link_libraries(${EXECUTABLE} gcc)
	# target_link_libraries(${EXECUTABLE} gdi32)
	# target_link_libraries(${EXECUTABLE} ws2_32)
	# target_link_libraries(${EXECUTABLE} mswsock)
	# target_link_libraries(${EXECUTABLE} shlwapi)
	# target_link_libraries(${EXECUTABLE} iphlpapi)
	# target_link_libraries(${EXECUTABLE} boost_thread_win32-mt-s)
	# set(CMAKE_SHARED_LIBRARY_LINK_CXX_FLAGS)
# else ()
	# find_package(Threads REQUIRED)
	# target_link_libraries(${EXECUTABLE} ${CMAKE_THREAD_LIBS_INIT})
# endif ()

# LLVM specific commands
find_package(LLVM REQUIRED CONFIG)
include_directories(${LLVM_INCLUDE_DIRS})
add_definitions(${LLVM_DEFINITIONS})

llvm_map_components_to_libnames(LLVM_LIBS core support mcjit x86asmparser x86codegen)
target_link_libraries(${PROJECT_NAME} ${LLVM_LIBS})
# end of LLVM specific commands


# Boost
find_package(Boost REQUIRED)
include_directories(${Boost_INCLUDE_DIRS})


#install( TARGETS ${EXECUTABLE} ARCHIVE DESTINATION lib LIBRARY DESTINATION lib )
#install( FILES ${HEADERS} DESTINATION include/${EXECUTABLE} )


source_group("" FILES ${HEADERS})
source_group("" FILES ${SOURCES})
set_property(TARGET ${PROJECT_NAME} PROPERTY FOLDER "libs")
