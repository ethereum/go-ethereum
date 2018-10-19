set(LIBRARY ubqhash)

if (CPPETHEREUM)
	set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -fPIC")
endif ()

set(CMAKE_BUILD_TYPE Release)

if (NOT MSVC)
	set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -std=gnu99")
endif()

set(FILES 	util.h
          	io.c
          	internal.c
          	ubqhash.h
          	endian.h
          	compiler.h
          	fnv.h
          	data_sizes.h)

if (MSVC)
	list(APPEND FILES util_win32.c io_win32.c mmap_win32.c)
else()
	list(APPEND FILES io_posix.c)
endif()

if (NOT CRYPTOPP_FOUND)
	find_package(CryptoPP 7.0.0)
endif()

if (CRYPTOPP_FOUND)
	add_definitions(-DWITH_CRYPTOPP)
	include_directories( ${CRYPTOPP_INCLUDE_DIRS} )
	list(APPEND FILES sha3_cryptopp.cpp sha3_cryptopp.h blake2_cryptopp.cpp blake2_cryptopp.h)
else()
	list(APPEND FILES sha3.c sha3.h blake2b-ref.c blake2-impl.h blake2.h)
endif()

add_library(${LIBRARY} ${FILES})

if (CRYPTOPP_FOUND)
	TARGET_LINK_LIBRARIES(${LIBRARY} ${CRYPTOPP_LIBRARIES})
endif()
