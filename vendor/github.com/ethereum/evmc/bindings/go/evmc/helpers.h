/* EVMC: Ethereum Client-VM Connector API.
 * Copyright 2018 The EVMC Authors.
 * Licensed under the Apache License, Version 2.0. See the LICENSE file.
 */

/**
 * EVMC Helpers
 *
 * A collection of helper functions for invoking a VM instance methods.
 * These are convenient for languages where invoking function pointers
 * is "ugly" or impossible (such as Go).
 *
 * It also contains helpers (overloaded operators) for using EVMC types effectively in C++.
 *
 * @defgroup helpers EVMC Helpers
 * @{
 */
#pragma once

#include <evmc/evmc.h>

/**
 * Returns true if the VM instance has a compatible ABI version.
 */
static inline int evmc_is_abi_compatible(struct evmc_instance* instance)
{
    return instance->abi_version == EVMC_ABI_VERSION;
}

/**
 * Returns the name of the VM instance.
 */
static inline const char* evmc_vm_name(struct evmc_instance* instance)
{
    return instance->name;
}

/**
 * Returns the version of the VM instance.
 */
static inline const char* evmc_vm_version(struct evmc_instance* instance)
{
    return instance->version;
}

/**
 * Checks if the VM instance has the given capability.
 *
 * @see evmc_get_capabilities_fn
 */
static inline bool evmc_vm_has_capability(struct evmc_instance* vm,
                                          enum evmc_capabilities capability)
{
    return (vm->get_capabilities(vm) & (evmc_capabilities_flagset)capability) != 0;
}

/**
 * Destroys the VM instance.
 *
 * @see evmc_destroy_fn
 */
static inline void evmc_destroy(struct evmc_instance* instance)
{
    instance->destroy(instance);
}

/**
 * Sets the option for the VM instance, if the feature is supported by the VM.
 *
 * @see evmc_set_option_fn
 */
static inline enum evmc_set_option_result evmc_set_option(struct evmc_instance* instance,
                                                          char const* name,
                                                          char const* value)
{
    if (instance->set_option)
        return instance->set_option(instance, name, value);
    return EVMC_SET_OPTION_INVALID_NAME;
}

/**
 * Sets the tracer callback for the VM instance, if the feature is supported by the VM.
 *
 * @see evmc_set_tracer_fn
 */
static inline void evmc_set_tracer(struct evmc_instance* instance,
                                   evmc_trace_callback callback,
                                   struct evmc_tracer_context* context)
{
    if (instance->set_tracer)
        instance->set_tracer(instance, callback, context);
}

/**
 * Executes code in the VM instance.
 *
 * @see evmc_execute_fn.
 */
static inline struct evmc_result evmc_execute(struct evmc_instance* instance,
                                              struct evmc_context* context,
                                              enum evmc_revision rev,
                                              const struct evmc_message* msg,
                                              uint8_t const* code,
                                              size_t code_size)
{
    return instance->execute(instance, context, rev, msg, code, code_size);
}

/**
 * Releases the resources allocated to the execution result.
 *
 * @see evmc_release_result_fn
 */
static inline void evmc_release_result(struct evmc_result* result)
{
    result->release(result);
}


/**
 * Helpers for optional storage of evmc_result.
 *
 * In some contexts (i.e. evmc_result::create_address is unused) objects of
 * type evmc_result contains a memory storage that MAY be used by the object
 * owner. This group defines helper types and functions for accessing
 * the optional storage.
 *
 * @defgroup result_optional_storage Result Optional Storage
 * @{
 */

/**
 * The union representing evmc_result "optional storage".
 *
 * The evmc_result struct contains 24 bytes of optional storage that can be
 * reused by the object creator if the object does not contain
 * evmc_result::create_address.
 *
 * A VM implementation MAY use this memory to keep additional data
 * when returning result from evmc_execute_fn().
 * The host application MAY use this memory to keep additional data
 * when returning result of performed calls from evmc_call_fn().
 *
 * @see evmc_get_optional_storage(), evmc_get_const_optional_storage().
 */
union evmc_result_optional_storage
{
    uint8_t bytes[24]; /**< 24 bytes of optional storage. */
    void* pointer;     /**< Optional pointer. */
};

/** Provides read-write access to evmc_result "optional storage". */
static inline union evmc_result_optional_storage* evmc_get_optional_storage(
    struct evmc_result* result)
{
    return (union evmc_result_optional_storage*)&result->create_address;
}

/** Provides read-only access to evmc_result "optional storage". */
static inline const union evmc_result_optional_storage* evmc_get_const_optional_storage(
    const struct evmc_result* result)
{
    return (const union evmc_result_optional_storage*)&result->create_address;
}

/** @} */

/** @} */
