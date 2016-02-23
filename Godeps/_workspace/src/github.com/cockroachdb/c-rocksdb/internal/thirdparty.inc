# Edit definitions below to specify paths to include files and libraries of all 3rd party libraries

#
# Edit these lines to set defaults for use of external libraries
#
set(USE_GFLAGS_DEFAULT 0)        # GFLAGS is disabled by default, enable with -DGFLAGS=1 cmake command line agrument
set(USE_SNAPPY_DEFAULT 0)        # SNAPPY is disabled by default, enable with -DSNAPPY=1 cmake command line agrument
set(USE_LZ4_DEFAULT 0)           # LZ4 is disabled by default, enable with -DLZ4=1 cmake command line agrument
set(USE_ZLIB_DEFAULT 0)          # ZLIB is disabled by default, enable with -DZLIB=1 cmake command line agrument
set(USE_JEMALLOC_DEFAULT 0)      # JEMALLOC is disabled by default, enable with -DJEMALLOC=1 cmake command line agrument

#
# This example assumes all the libraries locate in directories under THIRDPARTY_HOME environment variable
# Set environment variable THIRDPARTY_HOME to point to your third party libraries home (Unix style dir separators)
# or change the paths below to reflect where the libraries actually reside
#
set (THIRDPARTY_LIBS "")         # Initialization, don't touch

#
# Edit these 4 lines to define paths to GFLAGS
#
set(GFLAGS_HOME $ENV{THIRDPARTY_HOME}/Gflags.Library)
set(GFLAGS_INCLUDE ${GFLAGS_HOME}/inc/include)
set(GFLAGS_LIB_DEBUG ${GFLAGS_HOME}/bin/debug/amd64/gflags.lib)
set(GFLAGS_LIB_RELEASE ${GFLAGS_HOME}/bin/retail/amd64/gflags.lib)

# ================================================== GFLAGS ==================================================
#
# Don't touch these lines
#
if (DEFINED GFLAGS)
  set(USE_GFLAGS ${GFLAGS})
else ()
  set(USE_GFLAGS ${USE_GFLAGS_DEFAULT})
endif ()

if (${USE_GFLAGS} EQUAL 1)
  message("GFLAGS library is enabled")
  set(GFLAGS_CXX_FLAGS -DGFLAGS=gflags)
  set(GFLAGS_LIBS debug ${GFLAGS_LIB_DEBUG} optimized ${GFLAGS_LIB_RELEASE})

  add_definitions(${GFLAGS_CXX_FLAGS})
  include_directories(${GFLAGS_INCLUDE})
  set (THIRDPARTY_LIBS ${THIRDPARTY_LIBS} ${GFLAGS_LIBS})
else ()
  message("GFLAGS library is disabled")
endif ()

# ================================================== SNAPPY ==================================================
#
# Edit these 4 lines to define paths to Snappy
#
set(SNAPPY_HOME $ENV{THIRDPARTY_HOME}/Snappy.Library)
set(SNAPPY_INCLUDE ${SNAPPY_HOME}/inc/inc)
set(SNAPPY_LIB_DEBUG ${SNAPPY_HOME}/bin/debug/amd64/snappy.lib)
set(SNAPPY_LIB_RELEASE ${SNAPPY_HOME}/bin/retail/amd64/snappy.lib)

#
# Don't touch these lines
#
if (DEFINED SNAPPY)
  set(USE_SNAPPY ${SNAPPY})
else ()
  set(USE_SNAPPY ${USE_SNAPPY_DEFAULT})
endif ()

if (${USE_SNAPPY} EQUAL 1)
  message("SNAPPY library is enabled")
  set(SNAPPY_CXX_FLAGS -DSNAPPY)
  set(SNAPPY_LIBS debug ${SNAPPY_LIB_DEBUG} optimized ${SNAPPY_LIB_RELEASE})

  add_definitions(${SNAPPY_CXX_FLAGS})
  include_directories(${SNAPPY_INCLUDE})
  set (THIRDPARTY_LIBS ${THIRDPARTY_LIBS} ${SNAPPY_LIBS})
else ()
  message("SNAPPY library is disabled")
endif ()

# ================================================== LZ4 ==================================================
#
# Edit these 4 lines to define paths to LZ4
#
set(LZ4_HOME $ENV{THIRDPARTY_HOME}/LZ4.Library)
set(LZ4_INCLUDE ${LZ4_HOME}/inc/include)
set(LZ4_LIB_DEBUG ${LZ4_HOME}/bin/debug/amd64/lz4.lib)
set(LZ4_LIB_RELEASE ${LZ4_HOME}/bin/retail/amd64/lz4.lib)

#
# Don't touch these lines
#
if (DEFINED LZ4)
  set(USE_LZ4 ${LZ4})
else ()
  set(USE_LZ4 ${USE_LZ4_DEFAULT})
endif ()

if (${USE_LZ4} EQUAL 1)
  message("LZ4 library is enabled")
  set(LZ4_CXX_FLAGS -DLZ4)
  set(LZ4_LIBS debug ${LZ4_LIB_DEBUG} optimized ${LZ4_LIB_RELEASE})

  add_definitions(${LZ4_CXX_FLAGS})
  include_directories(${LZ4_INCLUDE})
  set (THIRDPARTY_LIBS ${THIRDPARTY_LIBS} ${LZ4_LIBS})
else ()
  message("LZ4 library is disabled")
endif ()

# ================================================== ZLIB ==================================================
#
# Edit these 4 lines to define paths to ZLIB
#
set(ZLIB_HOME $ENV{THIRDPARTY_HOME}/ZLIB.Library)
set(ZLIB_INCLUDE ${ZLIB_HOME}/inc/include)
set(ZLIB_LIB_DEBUG ${ZLIB_HOME}/bin/debug/amd64/zlib.lib)
set(ZLIB_LIB_RELEASE ${ZLIB_HOME}/bin/retail/amd64/zlib.lib)

#
# Don't touch these lines
#
if (DEFINED ZLIB)
  set(USE_ZLIB ${ZLIB})
else ()
  set(USE_ZLIB ${USE_ZLIB_DEFAULT})
endif ()

if (${USE_ZLIB} EQUAL 1)
  message("ZLIB library is enabled")
  set(ZLIB_CXX_FLAGS -DZLIB)
  set(ZLIB_LIBS debug ${ZLIB_LIB_DEBUG} optimized ${ZLIB_LIB_RELEASE})

  add_definitions(${ZLIB_CXX_FLAGS})
  include_directories(${ZLIB_INCLUDE})
  set (THIRDPARTY_LIBS ${THIRDPARTY_LIBS} ${ZLIB_LIBS})
else ()
  message("ZLIB library is disabled")
endif ()

#
# Edit these 4 lines to define paths to Jemalloc
#
set(JEMALLOC_HOME $ENV{THIRDPARTY_HOME}/Jemalloc.Library)
set(JEMALLOC_INCLUDE ${JEMALLOC_HOME}/inc/include)
set(JEMALLOC_LIB_DEBUG ${JEMALLOC_HOME}/bin/debug/amd64/jemalloc.lib)
set(JEMALLOC_LIB_RELEASE ${JEMALLOC_HOME}/bin/retail/amd64/jemalloc.lib)

# ================================================== JEMALLOC ==================================================
#
# Don't touch these lines
#
if (DEFINED JEMALLOC)
  set(USE_JEMALLOC ${JEMALLOC})
else ()
  set(USE_JEMALLOC ${USE_JEMALLOC_DEFAULT})
endif ()

if (${USE_JEMALLOC} EQUAL 1)
  message("JEMALLOC library is enabled")
  set(JEMALLOC_CXX_FLAGS -DJEMALLOC)
  set(JEMALLOC_LIBS debug ${JEMALLOC_LIB_DEBUG} optimized ${JEMALLOC_LIB_RELEASE})

  add_definitions(${JEMALLOC_CXX_FLAGS})
  include_directories(${JEMALLOC_INCLUDE})
  set (THIRDPARTY_LIBS ${THIRDPARTY_LIBS} ${JEMALLOC_LIBS})
  set (ARTIFACT_SUFFIX "_je")
else ()
  set (ARTIFACT_SUFFIX "")
  message("JEMALLOC library is disabled")
endif ()
