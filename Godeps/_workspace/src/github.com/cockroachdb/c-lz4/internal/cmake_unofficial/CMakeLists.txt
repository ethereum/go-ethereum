PROJECT(LZ4 C)
set(CPACK_PACKAGE_DESCRIPTION_SUMMARY "LZ4 compression library")
set(CPACK_PACKAGE_VERSION_MAJOR 1)
set(CPACK_PACKAGE_VERSION_MINOR 5)
set(CPACK_PACKAGE_VERSION_PATCH r128)
set(VERSION_STRING	" \"${CPACK_PACKAGE_VERSION_MAJOR}.${CPACK_PACKAGE_VERSION_MINOR}.${CPACK_PACKAGE_VERSION_PATCH}\" ")
include(CPack)

cmake_minimum_required (VERSION 2.6)
INCLUDE (CheckTypeSize)
check_type_size("void *" SIZEOF_VOID_P)
IF( ${SIZEOF_VOID_P} STREQUAL  "8" )
    set (CMAKE_SYSTEM_PROCESSOR "64bit")
    MESSAGE( STATUS "64 bit architecture detected size of void * is " ${SIZEOF_VOID_P})
ENDIF()

option(BUILD_TOOLS "Build the command line tools" ON)
option(BUILD_LIBS  "Build the libraries in addition to the tools" ON)

IF("${CMAKE_C_COMPILER_ID}" STREQUAL "GNU" OR
   "${CMAKE_C_COMPILER_ID}" STREQUAL "Clang")
  SET(GNU_COMPATIBLE_COMPILER 1)
ENDIF()

if(GNU_COMPATIBLE_COMPILER)
if(UNIX AND BUILD_LIBS)
    add_definitions(-fPIC)
endif()
endif()

set(LZ4_DIR ../lib/)
set(PRG_DIR ../programs/)
set(LZ4_SRCS_LIB ${LZ4_DIR}lz4.c ${LZ4_DIR}lz4hc.c ${LZ4_DIR}lz4.h ${LZ4_DIR}lz4hc.h ${LZ4_DIR}lz4frame.c ${LZ4_DIR}lz4frame.h ${LZ4_DIR}xxhash.c)
set(LZ4_SRCS ${LZ4_DIR}lz4frame.c ${LZ4_DIR}xxhash.c ${PRG_DIR}bench.c ${PRG_DIR}lz4cli.c ${PRG_DIR}lz4io.c)

if(BUILD_TOOLS AND NOT BUILD_LIBS)
    set(LZ4_SRCS ${LZ4_SRCS} ${LZ4_SRCS_LIB})
endif()

if(BUILD_TOOLS)
    add_executable(lz4 ${LZ4_SRCS})
    install(TARGETS lz4 RUNTIME DESTINATION "bin/")
endif()

if(BUILD_LIBS)    
    add_library(liblz4 ${LZ4_SRCS_LIB})
    
    set_target_properties(liblz4 PROPERTIES
    OUTPUT_NAME lz4
    SOVERSION "${CPACK_PACKAGE_VERSION_MAJOR}.${CPACK_PACKAGE_VERSION_MINOR}"
    )
        
    install(TARGETS liblz4
    LIBRARY DESTINATION lib
    ARCHIVE DESTINATION lib
    )

    install(FILES
    ${LZ4_DIR}/lz4.h
    ${LZ4_DIR}/lz4hc.h
    ${LZ4_DIR}/lz4frame.h
    DESTINATION include
    )
    
    if(BUILD_TOOLS)
        target_link_libraries(lz4 liblz4)
    endif()
endif()


#warnings

if(MSVC)
ADD_DEFINITIONS("-W4")
endif()
if(GNU_COMPATIBLE_COMPILER)
ADD_DEFINITIONS("-Wall")
endif()
if(CMAKE_COMPILER_IS_GNUCXX)
ADD_DEFINITIONS("-Wextra")
ADD_DEFINITIONS("-Wundef")
ADD_DEFINITIONS("-Wshadow")
ADD_DEFINITIONS("-Wcast-align")
ADD_DEFINITIONS("-Wstrict-prototypes")
endif(CMAKE_COMPILER_IS_GNUCXX)
if(GNU_COMPATIBLE_COMPILER AND
   (NOT CMAKE_SYSTEM_NAME MATCHES "SunOS"))
ADD_DEFINITIONS("-std=c99")
endif()
ADD_DEFINITIONS("-DLZ4_VERSION=\"${CPACK_PACKAGE_VERSION_PATCH}\"")
INCLUDE_DIRECTORIES (${LZ4_DIR})




