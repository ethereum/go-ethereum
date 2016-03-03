#.rst:
# FindPackageHandleStandardArgs
# -----------------------------
#
#
#
# FIND_PACKAGE_HANDLE_STANDARD_ARGS(<name> ...  )
#
# This function is intended to be used in FindXXX.cmake modules files.
# It handles the REQUIRED, QUIET and version-related arguments to
# find_package().  It also sets the <packagename>_FOUND variable.  The
# package is considered found if all variables <var1>...  listed contain
# valid results, e.g.  valid filepaths.
#
# There are two modes of this function.  The first argument in both
# modes is the name of the Find-module where it is called (in original
# casing).
#
# The first simple mode looks like this:
#
# ::
#
#     FIND_PACKAGE_HANDLE_STANDARD_ARGS(<name>
#       (DEFAULT_MSG|"Custom failure message") <var1>...<varN> )
#
# If the variables <var1> to <varN> are all valid, then
# <UPPERCASED_NAME>_FOUND will be set to TRUE.  If DEFAULT_MSG is given
# as second argument, then the function will generate itself useful
# success and error messages.  You can also supply a custom error
# message for the failure case.  This is not recommended.
#
# The second mode is more powerful and also supports version checking:
#
# ::
#
#     FIND_PACKAGE_HANDLE_STANDARD_ARGS(NAME
#       [FOUND_VAR <resultVar>]
#       [REQUIRED_VARS <var1>...<varN>]
#       [VERSION_VAR   <versionvar>]
#       [HANDLE_COMPONENTS]
#       [CONFIG_MODE]
#       [FAIL_MESSAGE "Custom failure message"] )
#
# In this mode, the name of the result-variable can be set either to
# either <UPPERCASED_NAME>_FOUND or <OriginalCase_Name>_FOUND using the
# FOUND_VAR option.  Other names for the result-variable are not
# allowed.  So for a Find-module named FindFooBar.cmake, the two
# possible names are FooBar_FOUND and FOOBAR_FOUND.  It is recommended
# to use the original case version.  If the FOUND_VAR option is not
# used, the default is <UPPERCASED_NAME>_FOUND.
#
# As in the simple mode, if <var1> through <varN> are all valid,
# <packagename>_FOUND will be set to TRUE.  After REQUIRED_VARS the
# variables which are required for this package are listed.  Following
# VERSION_VAR the name of the variable can be specified which holds the
# version of the package which has been found.  If this is done, this
# version will be checked against the (potentially) specified required
# version used in the find_package() call.  The EXACT keyword is also
# handled.  The default messages include information about the required
# version and the version which has been actually found, both if the
# version is ok or not.  If the package supports components, use the
# HANDLE_COMPONENTS option to enable handling them.  In this case,
# find_package_handle_standard_args() will report which components have
# been found and which are missing, and the <packagename>_FOUND variable
# will be set to FALSE if any of the required components (i.e.  not the
# ones listed after OPTIONAL_COMPONENTS) are missing.  Use the option
# CONFIG_MODE if your FindXXX.cmake module is a wrapper for a
# find_package(...  NO_MODULE) call.  In this case VERSION_VAR will be
# set to <NAME>_VERSION and the macro will automatically check whether
# the Config module was found.  Via FAIL_MESSAGE a custom failure
# message can be specified, if this is not used, the default message
# will be displayed.
#
# Example for mode 1:
#
# ::
#
#     find_package_handle_standard_args(LibXml2  DEFAULT_MSG
#       LIBXML2_LIBRARY LIBXML2_INCLUDE_DIR)
#
#
#
# LibXml2 is considered to be found, if both LIBXML2_LIBRARY and
# LIBXML2_INCLUDE_DIR are valid.  Then also LIBXML2_FOUND is set to
# TRUE.  If it is not found and REQUIRED was used, it fails with
# FATAL_ERROR, independent whether QUIET was used or not.  If it is
# found, success will be reported, including the content of <var1>.  On
# repeated Cmake runs, the same message won't be printed again.
#
# Example for mode 2:
#
# ::
#
#     find_package_handle_standard_args(LibXslt
#       FOUND_VAR LibXslt_FOUND
#       REQUIRED_VARS LibXslt_LIBRARIES LibXslt_INCLUDE_DIRS
#       VERSION_VAR LibXslt_VERSION_STRING)
#
# In this case, LibXslt is considered to be found if the variable(s)
# listed after REQUIRED_VAR are all valid, i.e.  LibXslt_LIBRARIES and
# LibXslt_INCLUDE_DIRS in this case.  The result will then be stored in
# LibXslt_FOUND .  Also the version of LibXslt will be checked by using
# the version contained in LibXslt_VERSION_STRING.  Since no
# FAIL_MESSAGE is given, the default messages will be printed.
#
# Another example for mode 2:
#
# ::
#
#     find_package(Automoc4 QUIET NO_MODULE HINTS /opt/automoc4)
#     find_package_handle_standard_args(Automoc4  CONFIG_MODE)
#
# In this case, FindAutmoc4.cmake wraps a call to find_package(Automoc4
# NO_MODULE) and adds an additional search directory for automoc4.  Here
# the result will be stored in AUTOMOC4_FOUND.  The following
# FIND_PACKAGE_HANDLE_STANDARD_ARGS() call produces a proper
# success/error message.

#=============================================================================
# Copyright 2007-2009 Kitware, Inc.
#
# Distributed under the OSI-approved BSD License (the "License");
# see accompanying file Copyright.txt for details.
#
# This software is distributed WITHOUT ANY WARRANTY; without even the
# implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
# See the License for more information.
#=============================================================================
# (To distribute this file outside of CMake, substitute the full
#  License text for the above reference.)

include(${CMAKE_CURRENT_LIST_DIR}/FindPackageMessage.cmake)
include(${CMAKE_CURRENT_LIST_DIR}/CMakeParseArguments.cmake)

# internal helper macro
macro(_FPHSA_FAILURE_MESSAGE _msg)
  if (${_NAME}_FIND_REQUIRED)
    message(FATAL_ERROR "${_msg}")
  else ()
    if (NOT ${_NAME}_FIND_QUIETLY)
      message(STATUS "${_msg}")
    endif ()
  endif ()
endmacro()


# internal helper macro to generate the failure message when used in CONFIG_MODE:
macro(_FPHSA_HANDLE_FAILURE_CONFIG_MODE)
  # <name>_CONFIG is set, but FOUND is false, this means that some other of the REQUIRED_VARS was not found:
  if(${_NAME}_CONFIG)
    _FPHSA_FAILURE_MESSAGE("${FPHSA_FAIL_MESSAGE}: missing: ${MISSING_VARS} (found ${${_NAME}_CONFIG} ${VERSION_MSG})")
  else()
    # If _CONSIDERED_CONFIGS is set, the config-file has been found, but no suitable version.
    # List them all in the error message:
    if(${_NAME}_CONSIDERED_CONFIGS)
      set(configsText "")
      list(LENGTH ${_NAME}_CONSIDERED_CONFIGS configsCount)
      math(EXPR configsCount "${configsCount} - 1")
      foreach(currentConfigIndex RANGE ${configsCount})
        list(GET ${_NAME}_CONSIDERED_CONFIGS ${currentConfigIndex} filename)
        list(GET ${_NAME}_CONSIDERED_VERSIONS ${currentConfigIndex} version)
        set(configsText "${configsText}    ${filename} (version ${version})\n")
      endforeach()
      if (${_NAME}_NOT_FOUND_MESSAGE)
        set(configsText "${configsText}    Reason given by package: ${${_NAME}_NOT_FOUND_MESSAGE}\n")
      endif()
      _FPHSA_FAILURE_MESSAGE("${FPHSA_FAIL_MESSAGE} ${VERSION_MSG}, checked the following files:\n${configsText}")

    else()
      # Simple case: No Config-file was found at all:
      _FPHSA_FAILURE_MESSAGE("${FPHSA_FAIL_MESSAGE}: found neither ${_NAME}Config.cmake nor ${_NAME_LOWER}-config.cmake ${VERSION_MSG}")
    endif()
  endif()
endmacro()


function(FIND_PACKAGE_HANDLE_STANDARD_ARGS _NAME _FIRST_ARG)

# set up the arguments for CMAKE_PARSE_ARGUMENTS and check whether we are in
# new extended or in the "old" mode:
  set(options  CONFIG_MODE  HANDLE_COMPONENTS)
  set(oneValueArgs  FAIL_MESSAGE  VERSION_VAR  FOUND_VAR)
  set(multiValueArgs REQUIRED_VARS)
  set(_KEYWORDS_FOR_EXTENDED_MODE  ${options} ${oneValueArgs} ${multiValueArgs} )
  list(FIND _KEYWORDS_FOR_EXTENDED_MODE "${_FIRST_ARG}" INDEX)

  if(${INDEX} EQUAL -1)
    set(FPHSA_FAIL_MESSAGE ${_FIRST_ARG})
    set(FPHSA_REQUIRED_VARS ${ARGN})
    set(FPHSA_VERSION_VAR)
  else()

    CMAKE_PARSE_ARGUMENTS(FPHSA "${options}" "${oneValueArgs}" "${multiValueArgs}"  ${_FIRST_ARG} ${ARGN})

    if(FPHSA_UNPARSED_ARGUMENTS)
      message(FATAL_ERROR "Unknown keywords given to FIND_PACKAGE_HANDLE_STANDARD_ARGS(): \"${FPHSA_UNPARSED_ARGUMENTS}\"")
    endif()

    if(NOT FPHSA_FAIL_MESSAGE)
      set(FPHSA_FAIL_MESSAGE  "DEFAULT_MSG")
    endif()
  endif()

# now that we collected all arguments, process them

  if("x${FPHSA_FAIL_MESSAGE}" STREQUAL "xDEFAULT_MSG")
    set(FPHSA_FAIL_MESSAGE "Could NOT find ${_NAME}")
  endif()

  # In config-mode, we rely on the variable <package>_CONFIG, which is set by find_package()
  # when it successfully found the config-file, including version checking:
  if(FPHSA_CONFIG_MODE)
    list(INSERT FPHSA_REQUIRED_VARS 0 ${_NAME}_CONFIG)
    list(REMOVE_DUPLICATES FPHSA_REQUIRED_VARS)
    set(FPHSA_VERSION_VAR ${_NAME}_VERSION)
  endif()

  if(NOT FPHSA_REQUIRED_VARS)
    message(FATAL_ERROR "No REQUIRED_VARS specified for FIND_PACKAGE_HANDLE_STANDARD_ARGS()")
  endif()

  list(GET FPHSA_REQUIRED_VARS 0 _FIRST_REQUIRED_VAR)

  string(TOUPPER ${_NAME} _NAME_UPPER)
  string(TOLOWER ${_NAME} _NAME_LOWER)

  if(FPHSA_FOUND_VAR)
    if(FPHSA_FOUND_VAR MATCHES "^${_NAME}_FOUND$"  OR  FPHSA_FOUND_VAR MATCHES "^${_NAME_UPPER}_FOUND$")
      set(_FOUND_VAR ${FPHSA_FOUND_VAR})
    else()
      message(FATAL_ERROR "The argument for FOUND_VAR is \"${FPHSA_FOUND_VAR}\", but only \"${_NAME}_FOUND\" and \"${_NAME_UPPER}_FOUND\" are valid names.")
    endif()
  else()
    set(_FOUND_VAR ${_NAME_UPPER}_FOUND)
  endif()

  # collect all variables which were not found, so they can be printed, so the
  # user knows better what went wrong (#6375)
  set(MISSING_VARS "")
  set(DETAILS "")
  # check if all passed variables are valid
  unset(${_FOUND_VAR})
  foreach(_CURRENT_VAR ${FPHSA_REQUIRED_VARS})
    if(NOT ${_CURRENT_VAR})
      set(${_FOUND_VAR} FALSE)
      set(MISSING_VARS "${MISSING_VARS} ${_CURRENT_VAR}")
    else()
      set(DETAILS "${DETAILS}[${${_CURRENT_VAR}}]")
    endif()
  endforeach()
  if(NOT "${${_FOUND_VAR}}" STREQUAL "FALSE")
    set(${_FOUND_VAR} TRUE)
  endif()

  # component handling
  unset(FOUND_COMPONENTS_MSG)
  unset(MISSING_COMPONENTS_MSG)

  if(FPHSA_HANDLE_COMPONENTS)
    foreach(comp ${${_NAME}_FIND_COMPONENTS})
      if(${_NAME}_${comp}_FOUND)

        if(NOT DEFINED FOUND_COMPONENTS_MSG)
          set(FOUND_COMPONENTS_MSG "found components: ")
        endif()
        set(FOUND_COMPONENTS_MSG "${FOUND_COMPONENTS_MSG} ${comp}")

      else()

        if(NOT DEFINED MISSING_COMPONENTS_MSG)
          set(MISSING_COMPONENTS_MSG "missing components: ")
        endif()
        set(MISSING_COMPONENTS_MSG "${MISSING_COMPONENTS_MSG} ${comp}")

        if(${_NAME}_FIND_REQUIRED_${comp})
          set(${_FOUND_VAR} FALSE)
          set(MISSING_VARS "${MISSING_VARS} ${comp}")
        endif()

      endif()
    endforeach()
    set(COMPONENT_MSG "${FOUND_COMPONENTS_MSG} ${MISSING_COMPONENTS_MSG}")
    set(DETAILS "${DETAILS}[c${COMPONENT_MSG}]")
  endif()

  # version handling:
  set(VERSION_MSG "")
  set(VERSION_OK TRUE)
  set(VERSION ${${FPHSA_VERSION_VAR}})

  # check with DEFINED here as the requested or found version may be "0"
  if (DEFINED ${_NAME}_FIND_VERSION)
    if(DEFINED ${FPHSA_VERSION_VAR})

      if(${_NAME}_FIND_VERSION_EXACT)       # exact version required
        # count the dots in the version string
        string(REGEX REPLACE "[^.]" "" _VERSION_DOTS "${VERSION}")
        # add one dot because there is one dot more than there are components
        string(LENGTH "${_VERSION_DOTS}." _VERSION_DOTS)
        if (_VERSION_DOTS GREATER ${_NAME}_FIND_VERSION_COUNT)
          # Because of the C++ implementation of find_package() ${_NAME}_FIND_VERSION_COUNT
          # is at most 4 here. Therefore a simple lookup table is used.
          if (${_NAME}_FIND_VERSION_COUNT EQUAL 1)
            set(_VERSION_REGEX "[^.]*")
          elseif (${_NAME}_FIND_VERSION_COUNT EQUAL 2)
            set(_VERSION_REGEX "[^.]*\\.[^.]*")
          elseif (${_NAME}_FIND_VERSION_COUNT EQUAL 3)
            set(_VERSION_REGEX "[^.]*\\.[^.]*\\.[^.]*")
          else ()
            set(_VERSION_REGEX "[^.]*\\.[^.]*\\.[^.]*\\.[^.]*")
          endif ()
          string(REGEX REPLACE "^(${_VERSION_REGEX})\\..*" "\\1" _VERSION_HEAD "${VERSION}")
          unset(_VERSION_REGEX)
          if (NOT ${_NAME}_FIND_VERSION VERSION_EQUAL _VERSION_HEAD)
            set(VERSION_MSG "Found unsuitable version \"${VERSION}\", but required is exact version \"${${_NAME}_FIND_VERSION}\"")
            set(VERSION_OK FALSE)
          else ()
            set(VERSION_MSG "(found suitable exact version \"${VERSION}\")")
          endif ()
          unset(_VERSION_HEAD)
        else ()
          if (NOT "${${_NAME}_FIND_VERSION}" VERSION_EQUAL "${VERSION}")
            set(VERSION_MSG "Found unsuitable version \"${VERSION}\", but required is exact version \"${${_NAME}_FIND_VERSION}\"")
            set(VERSION_OK FALSE)
          else ()
            set(VERSION_MSG "(found suitable exact version \"${VERSION}\")")
          endif ()
        endif ()
        unset(_VERSION_DOTS)

      else()     # minimum version specified:
        if ("${${_NAME}_FIND_VERSION}" VERSION_GREATER "${VERSION}")
          set(VERSION_MSG "Found unsuitable version \"${VERSION}\", but required is at least \"${${_NAME}_FIND_VERSION}\"")
          set(VERSION_OK FALSE)
        else ()
          set(VERSION_MSG "(found suitable version \"${VERSION}\", minimum required is \"${${_NAME}_FIND_VERSION}\")")
        endif ()
      endif()

    else()

      # if the package was not found, but a version was given, add that to the output:
      if(${_NAME}_FIND_VERSION_EXACT)
         set(VERSION_MSG "(Required is exact version \"${${_NAME}_FIND_VERSION}\")")
      else()
         set(VERSION_MSG "(Required is at least version \"${${_NAME}_FIND_VERSION}\")")
      endif()

    endif()
  else ()
    if(VERSION)
      set(VERSION_MSG "(found version \"${VERSION}\")")
    endif()
  endif ()

  if(VERSION_OK)
    set(DETAILS "${DETAILS}[v${VERSION}(${${_NAME}_FIND_VERSION})]")
  else()
    set(${_FOUND_VAR} FALSE)
  endif()


  # print the result:
  if (${_FOUND_VAR})
    FIND_PACKAGE_MESSAGE(${_NAME} "Found ${_NAME}: ${${_FIRST_REQUIRED_VAR}} ${VERSION_MSG} ${COMPONENT_MSG}" "${DETAILS}")
  else ()

    if(FPHSA_CONFIG_MODE)
      _FPHSA_HANDLE_FAILURE_CONFIG_MODE()
    else()
      if(NOT VERSION_OK)
        _FPHSA_FAILURE_MESSAGE("${FPHSA_FAIL_MESSAGE}: ${VERSION_MSG} (found ${${_FIRST_REQUIRED_VAR}})")
      else()
        _FPHSA_FAILURE_MESSAGE("${FPHSA_FAIL_MESSAGE} (missing: ${MISSING_VARS}) ${VERSION_MSG}")
      endif()
    endif()

  endif ()

  set(${_FOUND_VAR} ${${_FOUND_VAR}} PARENT_SCOPE)

endfunction()
