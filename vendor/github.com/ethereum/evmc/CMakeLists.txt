# EVMC: Ethereum Client-VM Connector API.
# Copyright 2018 The EVMC Authors.
# Licensed under the Apache License, Version 2.0. See the LICENSE file.

cmake_minimum_required(VERSION 3.5)

if(TARGET evmc)
    # The evmc library has been already created (probably by other submodule).
    return()
endif()

option(EVMC_TESTING "Build EVMC tests and test tools" OFF)
option(EVMC_EXAMPLES "Build EVMC examples" ${EVMC_TESTING})
option(HUNTER_ENABLED "Enable Hunter package manager support" "${EVMC_TESTING}")

include(cmake/cable/bootstrap.cmake)
include(CableBuildType)
include(CableCompilerSettings)
include(CMakePackageConfigHelpers)
include(HunterGate)
include(GNUInstallDirs)
include(defaults/HunterCacheServers)

include(HunterConfig)

project(evmc)
set(PROJECT_VERSION 6.0.0)

cable_set_build_type(DEFAULT Release CONFIGURATION_TYPES Debug Release)

cable_configure_compiler(NO_STACK_PROTECTION)

set(include_dir ${PROJECT_SOURCE_DIR}/include)

add_library(evmc INTERFACE)
add_library(evmc::evmc ALIAS evmc)
target_include_directories(evmc INTERFACE $<BUILD_INTERFACE:${include_dir}>$<INSTALL_INTERFACE:include>)
install(TARGETS evmc EXPORT evmcTargets)

install(DIRECTORY include/evmc DESTINATION ${CMAKE_INSTALL_INCLUDEDIR})

add_subdirectory(lib)

if(EVMC_TESTING)
    enable_testing()
    add_subdirectory(test)
endif()

if(EVMC_EXAMPLES OR EVMC_TESTING)
    add_subdirectory(examples)
endif()

write_basic_package_version_file(evmcConfigVersion.cmake COMPATIBILITY ExactVersion)

configure_package_config_file(
    cmake/Config.cmake.in
    evmcConfig.cmake
    INSTALL_DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/evmc
)

install(
    EXPORT evmcTargets
    NAMESPACE evmc::
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/evmc
)
install(
    FILES
    ${CMAKE_CURRENT_BINARY_DIR}/evmcConfig.cmake
    ${CMAKE_CURRENT_BINARY_DIR}/evmcConfigVersion.cmake
    DESTINATION ${CMAKE_INSTALL_LIBDIR}/cmake/evmc
)

if(WIN32)
    set(CPACK_GENERATOR ZIP)
else()
    set(CPACK_GENERATOR TGZ)
endif()
string(TOLOWER ${CMAKE_SYSTEM_NAME} system_name)
set(CPACK_PACKAGE_FILE_NAME ${PROJECT_NAME}-${PROJECT_VERSION}-${system_name})
set(CPACK_PACKAGE_CHECKSUM SHA256)
set(CPACK_INCLUDE_TOPLEVEL_DIRECTORY FALSE)
include(CPack)
