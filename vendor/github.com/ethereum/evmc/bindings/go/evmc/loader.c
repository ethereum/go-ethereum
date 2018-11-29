/* EVMC: Ethereum Client-VM Connector API.
 * Copyright 2018 The EVMC Authors.
 * Licensed under the Apache License, Version 2.0. See the LICENSE file.
 */

#include <evmc/loader.h>

#include <evmc/evmc.h>
#include <evmc/helpers.h>

#include <stdint.h>
#include <string.h>

#if _WIN32
#include <Windows.h>
#define DLL_HANDLE HMODULE
#define DLL_OPEN(filename) LoadLibrary(filename)
#define DLL_CLOSE(handle) FreeLibrary(handle)
#define DLL_GET_CREATE_FN(handle, name) (evmc_create_fn)(uintptr_t) GetProcAddress(handle, name)
#define HAVE_STRCPY_S 1
#else
#include <dlfcn.h>
#define DLL_HANDLE void*
#define DLL_OPEN(filename) dlopen(filename, RTLD_LAZY)
#define DLL_CLOSE(handle) dlclose(handle)
#define DLL_GET_CREATE_FN(handle, name) (evmc_create_fn)(uintptr_t) dlsym(handle, name)
#define HAVE_STRCPY_S 0
#endif

#define PATH_MAX_LENGTH 4096

#if !HAVE_STRCPY_S
static void strcpy_s(char* dest, size_t destsz, const char* src)
{
    size_t len = strlen(src);
    if (len > destsz - 1)
        len = destsz - 1;
    memcpy(dest, src, len);
    dest[len] = 0;
}
#endif


evmc_create_fn evmc_load(const char* filename, enum evmc_loader_error_code* error_code)
{
    enum evmc_loader_error_code ec = EVMC_LOADER_SUCCESS;
    evmc_create_fn create_fn = NULL;

    if (!filename)
    {
        ec = EVMC_LOADER_INVALID_ARGUMENT;
        goto exit;
    }

    const size_t length = strlen(filename);
    if (length == 0 || length > PATH_MAX_LENGTH)
    {
        ec = EVMC_LOADER_INVALID_ARGUMENT;
        goto exit;
    }

    DLL_HANDLE handle = DLL_OPEN(filename);
    if (!handle)
    {
        ec = EVMC_LOADER_CANNOT_OPEN;
        goto exit;
    }

    // Create name buffer with the prefix.
    const char prefix[] = "evmc_create_";
    const size_t prefix_length = strlen(prefix);
    char prefixed_name[sizeof(prefix) + PATH_MAX_LENGTH];
    strcpy_s(prefixed_name, sizeof(prefixed_name), prefix);

    // Find filename in the path.
    const char* sep_pos = strrchr(filename, '/');
#if _WIN32
    // On Windows check also Windows classic path separator.
    const char* sep_pos_windows = strrchr(filename, '\\');
    sep_pos = sep_pos_windows > sep_pos ? sep_pos_windows : sep_pos;
#endif
    const char* name_pos = sep_pos ? sep_pos + 1 : filename;

    // Skip "lib" prefix if present.
    const char lib_prefix[] = "lib";
    const size_t lib_prefix_length = strlen(lib_prefix);
    if (strncmp(name_pos, lib_prefix, lib_prefix_length) == 0)
        name_pos += lib_prefix_length;

    char* base_name = prefixed_name + prefix_length;
    strcpy_s(base_name, PATH_MAX_LENGTH, name_pos);

    // Trim the file extension.
    char* ext_pos = strrchr(prefixed_name, '.');
    if (ext_pos)
        *ext_pos = 0;

    // Replace all "-" with "_".
    char* dash_pos = base_name;
    while ((dash_pos = strchr(dash_pos, '-')) != NULL)
        *dash_pos++ = '_';

    // Search for the built function name.
    while ((create_fn = DLL_GET_CREATE_FN(handle, prefixed_name)) == NULL)
    {
        // Shorten the base name by skipping the `word_` segment.
        const char* shorter_name_pos = strchr(base_name, '_');
        if (!shorter_name_pos)
            break;

        memmove(base_name, shorter_name_pos + 1, strlen(shorter_name_pos) + 1);
    }

    if (!create_fn)
        create_fn = DLL_GET_CREATE_FN(handle, "evmc_create");

    if (!create_fn)
    {
        DLL_CLOSE(handle);
        ec = EVMC_LOADER_SYMBOL_NOT_FOUND;
    }

exit:
    if (error_code)
        *error_code = ec;
    return create_fn;
}

struct evmc_instance* evmc_load_and_create(const char* filename,
                                           enum evmc_loader_error_code* error_code)
{
    evmc_create_fn create_fn = evmc_load(filename, error_code);

    if (!create_fn)
        return NULL;

    struct evmc_instance* instance = create_fn();
    if (!instance)
    {
        *error_code = EVMC_LOADER_INSTANCE_CREATION_FAILURE;
        return NULL;
    }

    if (!evmc_is_abi_compatible(instance))
    {
        *error_code = EVMC_LOADER_ABI_VERSION_MISMATCH;
        return NULL;
    }

    return instance;
}
