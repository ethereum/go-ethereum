/* EVMC: Ethereum Client-VM Connector API.
 * Copyright 2018 The EVMC Authors.
 * Licensed under the Apache License, Version 2.0. See the LICENSE file.
 */

/**
 * EVMC Loader Library
 *
 * The EVMC Loader Library supports loading VMs implemented as Dynamically Loaded Libraries
 * (DLLs, shared objects).
 *
 * @defgroup loader EVMC Loader
 * @{
 */
#pragma once

#if __cplusplus
extern "C" {
#endif

/** The function pointer type for EVMC create functions. */
typedef struct evmc_instance* (*evmc_create_fn)(void);

/** Error codes for the EVMC loader. */
enum evmc_loader_error_code
{
    /** The loader succeeded. */
    EVMC_LOADER_SUCCESS = 0,

    /** The loader cannot open the given file name. */
    EVMC_LOADER_CANNOT_OPEN = 1,

    /** The VM create function not found. */
    EVMC_LOADER_SYMBOL_NOT_FOUND = 2,

    /** The invalid argument value provided. */
    EVMC_LOADER_INVALID_ARGUMENT = 3,

    /** The creation of a VM instance has failed. */
    EVMC_LOADER_INSTANCE_CREATION_FAILURE = 4,

    /** The ABI version of the VM instance has mismatched. */
    EVMC_LOADER_ABI_VERSION_MISMATCH = 5,
};

/**
 * Dynamically loads the shared object (DLL) with an EVM implementation.
 *
 * This function tries to open a DLL at the given `filename`. On UNIX-like systems dlopen() function
 * is used. On Windows LoadLibrary() function is used.
 *
 * If the file does not exist or is not a valid shared library the ::EVMC_LOADER_CANNOT_OPEN error
 * code is signaled and NULL is returned.
 *
 * After the DLL is successfully loaded the function tries to find the EVM create function in the
 * library. The `filename` is used to guess the EVM name and the name of the create function.
 * The create function name is constructed by the following rules. Consider example path:
 * "/ethereum/libexample-interpreter.so".
 * - the filename is taken from the path:
 *   "libexample-interpreter.so",
 * - the "lib" prefix and file extension are stripped from the name:
 *   "example-interpreter"
 * - all "-" are replaced with "_" to construct _base name_:
 *   "example_interpreter",
 * - the function name "evmc_create_" + _base name_ is searched in the library:
 *   "evmc_create_example_interpreter",
 * - if function not found, the _base name_ is shorten by skipping the first word separated by "_":
 *   "interpreter",
 * - then, the function of the shorter name "evmc_create_" + _base name_ is searched in the library:
 *   "evmc_create_interpreter",
 * - the name shortening continues until a function is found or the name cannot be shorten more,
 * - lastly, when no function found, the function name "evmc_create" is searched in the library.
 *
 * If the create function is found in the library, the pointer to the function is returned.
 * Otherwise, the ::EVMC_LOADER_SYMBOL_NOT_FOUND error code is signaled and NULL is returned.
 *
 * It is safe to call this function with the same filename argument multiple times
 * (the DLL is not going to be loaded multiple times).
 *
 * @param filename    The null terminated path (absolute or relative) to the shared library
 *                    containing the EVM implementation. If the value is NULL, an empty C-string
 *                    or longer than the path maximum length the ::EVMC_LOADER_INVALID_ARGUMENT is
 *                    signaled.
 * @param error_code  The pointer to the error code. If not NULL the value is set to
 *                    ::EVMC_LOADER_SUCCESS on success or any other error code as described above.
 * @return            The pointer to the EVM create function or NULL.
 */
evmc_create_fn evmc_load(const char* filename, enum evmc_loader_error_code* error_code);

/**
 * Dynamically loads the VM DLL and creates the VM instance.
 *
 * This is a macro for creating the VM instance with the function returned from evmc_load().
 * The function signals the same errors as evmc_load() and additionally:
 *  - ::EVMC_LOADER_INSTANCE_CREATION_FAILURE when the create function returns NULL,
 *  - ::EVMC_LOADER_ABI_VERSION_MISMATCH when the created VM instance has ABI version different
 *  from the ABI version of this library (::EVMC_ABI_VERSION).
 *
 * It is safe to call this function with the same filename argument multiple times:
 * the DLL is not going to be loaded multiple times, but the function will return new VM instance
 * each time.
 */
struct evmc_instance* evmc_load_and_create(const char* filename,
                                           enum evmc_loader_error_code* error_code);

#if __cplusplus
}
#endif

/** @} */
