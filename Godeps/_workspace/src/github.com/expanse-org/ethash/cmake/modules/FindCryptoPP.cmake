# Module for locating the Crypto++ encryption library.
#
# Customizable variables:
#   CRYPTOPP_ROOT_DIR
#     This variable points to the CryptoPP root directory. On Windows the
#     library location typically will have to be provided explicitly using the
#     -D command-line option. The directory should include the include/cryptopp,
#     lib and/or bin sub-directories.
#
# Read-only variables:
#   CRYPTOPP_FOUND
#     Indicates whether the library has been found.
#
#   CRYPTOPP_INCLUDE_DIRS
#     Points to the CryptoPP include directory.
#
#   CRYPTOPP_LIBRARIES
#     Points to the CryptoPP libraries that should be passed to
#     target_link_libararies.
#
#
# Copyright (c) 2012 Sergiu Dotenco
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

INCLUDE (FindPackageHandleStandardArgs)

FIND_PATH (CRYPTOPP_ROOT_DIR
  NAMES cryptopp/cryptlib.h include/cryptopp/cryptlib.h
  PATHS ENV CRYPTOPPROOT
  DOC "CryptoPP root directory")

# Re-use the previous path:
FIND_PATH (CRYPTOPP_INCLUDE_DIR
  NAMES cryptopp/cryptlib.h
  HINTS ${CRYPTOPP_ROOT_DIR}
  PATH_SUFFIXES include
  DOC "CryptoPP include directory")

FIND_LIBRARY (CRYPTOPP_LIBRARY_DEBUG
  NAMES cryptlibd cryptoppd
  HINTS ${CRYPTOPP_ROOT_DIR}
  PATH_SUFFIXES lib
  DOC "CryptoPP debug library")

FIND_LIBRARY (CRYPTOPP_LIBRARY_RELEASE
  NAMES cryptlib cryptopp
  HINTS ${CRYPTOPP_ROOT_DIR}
  PATH_SUFFIXES lib
  DOC "CryptoPP release library")

IF (CRYPTOPP_LIBRARY_DEBUG AND CRYPTOPP_LIBRARY_RELEASE)
  SET (CRYPTOPP_LIBRARY
    optimized ${CRYPTOPP_LIBRARY_RELEASE}
    debug ${CRYPTOPP_LIBRARY_DEBUG} CACHE DOC "CryptoPP library")
ELSEIF (CRYPTOPP_LIBRARY_RELEASE)
  SET (CRYPTOPP_LIBRARY ${CRYPTOPP_LIBRARY_RELEASE} CACHE DOC
    "CryptoPP library")
ENDIF (CRYPTOPP_LIBRARY_DEBUG AND CRYPTOPP_LIBRARY_RELEASE)

IF (CRYPTOPP_INCLUDE_DIR)
  SET (_CRYPTOPP_VERSION_HEADER ${CRYPTOPP_INCLUDE_DIR}/cryptopp/config.h)

  IF (EXISTS ${_CRYPTOPP_VERSION_HEADER})
    FILE (STRINGS ${_CRYPTOPP_VERSION_HEADER} _CRYPTOPP_VERSION_TMP REGEX
      "^#define CRYPTOPP_VERSION[ \t]+[0-9]+$")

    STRING (REGEX REPLACE
      "^#define CRYPTOPP_VERSION[ \t]+([0-9]+)" "\\1" _CRYPTOPP_VERSION_TMP
      ${_CRYPTOPP_VERSION_TMP})

    STRING (REGEX REPLACE "([0-9]+)[0-9][0-9]" "\\1" CRYPTOPP_VERSION_MAJOR
      ${_CRYPTOPP_VERSION_TMP})
    STRING (REGEX REPLACE "[0-9]([0-9])[0-9]" "\\1" CRYPTOPP_VERSION_MINOR
      ${_CRYPTOPP_VERSION_TMP})
    STRING (REGEX REPLACE "[0-9][0-9]([0-9])" "\\1" CRYPTOPP_VERSION_PATCH
      ${_CRYPTOPP_VERSION_TMP})

    SET (CRYPTOPP_VERSION_COUNT 3)
    SET (CRYPTOPP_VERSION
      ${CRYPTOPP_VERSION_MAJOR}.${CRYPTOPP_VERSION_MINOR}.${CRYPTOPP_VERSION_PATCH})
  ENDIF (EXISTS ${_CRYPTOPP_VERSION_HEADER})
ENDIF (CRYPTOPP_INCLUDE_DIR)

SET (CRYPTOPP_INCLUDE_DIRS ${CRYPTOPP_INCLUDE_DIR})
SET (CRYPTOPP_LIBRARIES ${CRYPTOPP_LIBRARY})

MARK_AS_ADVANCED (CRYPTOPP_INCLUDE_DIR CRYPTOPP_LIBRARY CRYPTOPP_LIBRARY_DEBUG
  CRYPTOPP_LIBRARY_RELEASE)

FIND_PACKAGE_HANDLE_STANDARD_ARGS (CryptoPP REQUIRED_VARS CRYPTOPP_ROOT_DIR
  CRYPTOPP_INCLUDE_DIR CRYPTOPP_LIBRARY VERSION_VAR CRYPTOPP_VERSION)