/**
 * EVMC: Ethereum Client-VM Connector API
 *
 * @copyright
 * Copyright 2018 The EVMC Authors.
 * Licensed under the Apache License, Version 2.0. See the LICENSE file.
 *
 * @defgroup EVMC EVMC
 * @{
 */
#ifndef EVMC_H
#define EVMC_H

#include <stdbool.h> /* Definition of bool, true and false. */
#include <stddef.h>  /* Definition of size_t. */
#include <stdint.h>  /* Definition of int64_t, uint64_t. */

#if __cplusplus
extern "C" {
#endif

/* BEGIN Python CFFI declarations */

enum
{
    /**
     * The EVMC ABI version number of the interface declared in this file.
     *
     * The EVMC ABI version always equals the major version number of the EVMC project.
     * The Host SHOULD check if the ABI versions match when dynamically loading VMs.
     *
     * @see @ref versioning
     */
    EVMC_ABI_VERSION = 6
};


/**
 * The fixed size array of 32 bytes.
 *
 * 32 bytes of data capable of storing e.g. 256-bit hashes.
 */
typedef struct evmc_bytes32
{
    /** The 32 bytes. */
    uint8_t bytes[32];
} evmc_bytes32;

/**
 * The alias for evmc_bytes32 to represent a big-endian 256-bit integer.
 */
typedef struct evmc_bytes32 evmc_uint256be;

/** Big-endian 160-bit hash suitable for keeping an Ethereum address. */
typedef struct evmc_address
{
    /** The 20 bytes of the hash. */
    uint8_t bytes[20];
} evmc_address;

/** The kind of call-like instruction. */
enum evmc_call_kind
{
    EVMC_CALL = 0,         /**< Request CALL. */
    EVMC_DELEGATECALL = 1, /**< Request DELEGATECALL. Valid since Homestead.
                                The value param ignored. */
    EVMC_CALLCODE = 2,     /**< Request CALLCODE. */
    EVMC_CREATE = 3,       /**< Request CREATE. */
    EVMC_CREATE2 = 4       /**< Request CREATE2. Valid since Constantinople.*/
};

/** The flags for ::evmc_message. */
enum evmc_flags
{
    EVMC_STATIC = 1 /**< Static call mode. */
};

/**
 * The message describing an EVM call,
 * including a zero-depth calls from a transaction origin.
 */
struct evmc_message
{
    /** The kind of the call. For zero-depth calls ::EVMC_CALL SHOULD be used. */
    enum evmc_call_kind kind;

    /**
     * Additional flags modifying the call execution behavior.
     * In the current version the only valid values are ::EVMC_STATIC or 0.
     */
    uint32_t flags;

    /** The call depth. */
    int32_t depth;

    /** The amount of gas for message execution. */
    int64_t gas;

    /** The destination of the message. */
    evmc_address destination;

    /** The sender of the message. */
    evmc_address sender;

    /**
     * The message input data.
     *
     * This MAY be NULL.
     */
    const uint8_t* input_data;

    /**
     * The size of the message input data.
     *
     * If input_data is NULL this MUST be 0.
     */
    size_t input_size;

    /**
     * The amount of Ether transferred with the message.
     */
    evmc_uint256be value;

    /**
     * The optional value used in new contract address construction.
     *
     * Ignored unless kind is EVMC_CREATE2.
     */
    evmc_bytes32 create2_salt;
};


/** The transaction and block data for execution. */
struct evmc_tx_context
{
    evmc_uint256be tx_gas_price;     /**< The transaction gas price. */
    evmc_address tx_origin;          /**< The transaction origin account. */
    evmc_address block_coinbase;     /**< The miner of the block. */
    int64_t block_number;            /**< The block number. */
    int64_t block_timestamp;         /**< The block timestamp. */
    int64_t block_gas_limit;         /**< The block gas limit. */
    evmc_uint256be block_difficulty; /**< The block difficulty. */
};

struct evmc_context;

/**
 * Get transaction context callback function.
 *
 *  This callback function is used by an EVM to retrieve the transaction and
 *  block context.
 *
 *  @param      context  The pointer to the Host execution context.
 *  @return              The transaction context.
 */
typedef struct evmc_tx_context (*evmc_get_tx_context_fn)(struct evmc_context* context);

/**
 * Get block hash callback function.
 *
 * This callback function is used by a VM to query the hash of the header of the given block.
 * If the information about the requested block is not available, then this is signalled by
 * returning null bytes.
 *
 * @param context  The pointer to the Host execution context.
 * @param number   The block number.
 * @return         The block hash or null bytes
 *                 if the information about the block is not available.
 */
typedef evmc_bytes32 (*evmc_get_block_hash_fn)(struct evmc_context* context, int64_t number);

/**
 * The execution status code.
 *
 * Successful execution is represented by ::EVMC_SUCCESS having value 0.
 *
 * Positive values represent failures defined by VM specifications with generic
 * ::EVMC_FAILURE code of value 1.
 *
 * Status codes with negative values represent VM internal errors
 * not provided by EVM specifications. These errors MUST not be passed back
 * to the caller. They MAY be handled by the Client in predefined manner
 * (see e.g. ::EVMC_REJECTED), otherwise internal errors are not recoverable.
 * The generic representant of errors is ::EVMC_INTERNAL_ERROR but
 * an EVM implementation MAY return negative status codes that are not defined
 * in the EVMC documentation.
 *
 * @note
 * In case new status codes are needed, please create an issue or pull request
 * in the EVMC repository (https://github.com/ethereum/evmc).
 */
enum evmc_status_code
{
    /** Execution finished with success. */
    EVMC_SUCCESS = 0,

    /** Generic execution failure. */
    EVMC_FAILURE = 1,

    /**
     * Execution terminated with REVERT opcode.
     *
     * In this case the amount of gas left MAY be non-zero and additional output
     * data MAY be provided in ::evmc_result.
     */
    EVMC_REVERT = 2,

    /** The execution has run out of gas. */
    EVMC_OUT_OF_GAS = 3,

    /**
     * The designated INVALID instruction has been hit during execution.
     *
     * The EIP-141 (https://github.com/ethereum/EIPs/blob/master/EIPS/eip-141.md)
     * defines the instruction 0xfe as INVALID instruction to indicate execution
     * abortion coming from high-level languages. This status code is reported
     * in case this INVALID instruction has been encountered.
     */
    EVMC_INVALID_INSTRUCTION = 4,

    /** An undefined instruction has been encountered. */
    EVMC_UNDEFINED_INSTRUCTION = 5,

    /**
     * The execution has attempted to put more items on the EVM stack
     * than the specified limit.
     */
    EVMC_STACK_OVERFLOW = 6,

    /** Execution of an opcode has required more items on the EVM stack. */
    EVMC_STACK_UNDERFLOW = 7,

    /** Execution has violated the jump destination restrictions. */
    EVMC_BAD_JUMP_DESTINATION = 8,

    /**
     * Tried to read outside memory bounds.
     *
     * An example is RETURNDATACOPY reading past the available buffer.
     */
    EVMC_INVALID_MEMORY_ACCESS = 9,

    /** Call depth has exceeded the limit (if any) */
    EVMC_CALL_DEPTH_EXCEEDED = 10,

    /** Tried to execute an operation which is restricted in static mode. */
    EVMC_STATIC_MODE_VIOLATION = 11,

    /**
     * A call to a precompiled or system contract has ended with a failure.
     *
     * An example: elliptic curve functions handed invalid EC points.
     */
    EVMC_PRECOMPILE_FAILURE = 12,

    /**
     * Contract validation has failed (e.g. due to EVM 1.5 jump validity,
     * Casper's purity checker or ewasm contract rules).
     */
    EVMC_CONTRACT_VALIDATION_FAILURE = 13,

    /**
     * An argument to a state accessing method has a value outside of the
     * accepted range of values.
     */
    EVMC_ARGUMENT_OUT_OF_RANGE = 14,

    /**
     * A WebAssembly `unreachable` instruction has been hit during execution.
     */
    EVMC_WASM_UNREACHABLE_INSTRUCTION = 15,

    /**
     * A WebAssembly trap has been hit during execution. This can be for many
     * reasons, including division by zero, validation errors, etc.
     */
    EVMC_WASM_TRAP = 16,

    /** EVM implementation generic internal error. */
    EVMC_INTERNAL_ERROR = -1,

    /**
     * The execution of the given code and/or message has been rejected
     * by the EVM implementation.
     *
     * This error SHOULD be used to signal that the EVM is not able to or
     * willing to execute the given code type or message.
     * If an EVM returns the ::EVMC_REJECTED status code,
     * the Client MAY try to execute it in other EVM implementation.
     * For example, the Client tries running a code in the EVM 1.5. If the
     * code is not supported there, the execution falls back to the EVM 1.0.
     */
    EVMC_REJECTED = -2
};

/* Forward declaration. */
struct evmc_result;

/**
 * Releases resources assigned to an execution result.
 *
 * This function releases memory (and other resources, if any) assigned to the
 * specified execution result making the result object invalid.
 *
 * @param result  The execution result which resources are to be released. The
 *                result itself it not modified by this function, but becomes
 *                invalid and user MUST discard it as well.
 *                This MUST NOT be NULL.
 *
 * @note
 * The result is passed by pointer to avoid (shallow) copy of the ::evmc_result
 * struct. Think of this as the best possible C language approximation to
 * passing objects by reference.
 */
typedef void (*evmc_release_result_fn)(const struct evmc_result* result);

/** The EVM code execution result. */
struct evmc_result
{
    /** The execution status code. */
    enum evmc_status_code status_code;

    /**
     * The amount of gas left after the execution.
     *
     *  If evmc_result::code is not ::EVMC_SUCCESS nor ::EVMC_REVERT
     *  the value MUST be 0.
     */
    int64_t gas_left;

    /**
     * The reference to output data.
     *
     *  The output contains data coming from RETURN opcode (iff evmc_result::code
     *  field is ::EVMC_SUCCESS) or from REVERT opcode.
     *
     *  The memory containing the output data is owned by EVM and has to be
     *  freed with evmc_result::release().
     *
     *  This MAY be NULL.
     */
    const uint8_t* output_data;

    /**
     * The size of the output data.
     *
     *  If output_data is NULL this MUST be 0.
     */
    size_t output_size;

    /**
     * The pointer to a function releasing all resources associated with
     *  the result object.
     *
     *  This function pointer is optional (MAY be NULL) and MAY be set by
     *  the EVM implementation. If set it MUST be used by the user to
     *  release memory and other resources associated with the result object.
     *  After the result resources are released the result object
     *  MUST NOT be used any more.
     *
     *  The suggested code pattern for releasing EVM results:
     *  @code
     *  struct evmc_result result = ...;
     *  if (result.release)
     *      result.release(&result);
     *  @endcode
     *
     *  @note
     *  It works similarly to C++ virtual destructor. Attaching the release
     *  function to the result itself allows EVM composition.
     */
    evmc_release_result_fn release;

    /**
     * The address of the contract created by CREATE opcode.
     *
     *  This field has valid value only if the result describes successful
     *  CREATE (evmc_result::status_code is ::EVMC_SUCCESS).
     */
    evmc_address create_address;

    /**
     * Reserved data that MAY be used by a evmc_result object creator.
     *
     *  This reserved 4 bytes together with 20 bytes from create_address form
     *  24 bytes of memory called "optional data" within evmc_result struct
     *  to be optionally used by the evmc_result object creator.
     *
     *  @see evmc_result_optional_data, evmc_get_optional_data().
     *
     *  Also extends the size of the evmc_result to 64 bytes (full cache line).
     */
    uint8_t padding[4];
};


/**
 * Check account existence callback function.
 *
 * This callback function is used by the VM to check if
 * there exists an account at given address.
 * @param context  The pointer to the Host execution context.
 * @param address  The address of the account the query is about.
 * @return         true if exists, false otherwise.
 */
typedef bool (*evmc_account_exists_fn)(struct evmc_context* context, const evmc_address* address);

/**
 * Get storage callback function.
 *
 * This callback function is used by a VM to query the given account storage entry.
 *
 * @param context  The Host execution context.
 * @param address  The address of the account.
 * @param key      The index of the account's storage entry.
 * @return         The storage value at the given storage key or null bytes
 *                 if the account does not exist.
 */
typedef evmc_bytes32 (*evmc_get_storage_fn)(struct evmc_context* context,
                                            const evmc_address* address,
                                            const evmc_bytes32* key);


/**
 * The effect of an attempt to modify a contract storage item.
 *
 * For the purpose of explaining the meaning of each element, the following
 * notation is used:
 * - 0 is zero value,
 * - X != 0 (X is any value other than 0),
 * - Y != X, Y != 0 (Y is any value other than X and 0),
 * - Z != Y (Z is any value other than Y),
 * - the "->" means the change from one value to another.
 */
enum evmc_storage_status
{
    /**
     * The value of a storage item has been left unchanged: 0 -> 0 and X -> X.
     */
    EVMC_STORAGE_UNCHANGED = 0,

    /**
     * The value of a storage item has been modified: X -> Y.
     */
    EVMC_STORAGE_MODIFIED = 1,

    /**
     * A storage item has been modified after being modified before: X -> Y -> Z.
     */
    EVMC_STORAGE_MODIFIED_AGAIN = 2,

    /**
     * A new storage item has been added: 0 -> X.
     */
    EVMC_STORAGE_ADDED = 3,

    /**
     * A storage item has been deleted: X -> 0.
     */
    EVMC_STORAGE_DELETED = 4
};


/**
 * Set storage callback function.
 *
 * This callback function is used by a VM to update the given account storage entry.
 * The VM MUST make sure that the account exists. This requirement is only a formality because
 * VM implementations only modify storage of the account of the current execution context
 * (i.e. referenced by evmc_message::destination).
 *
 * @param context  The pointer to the Host execution context.
 * @param address  The address of the account.
 * @param key      The index of the storage entry.
 * @param value    The value to be stored.
 * @return         The effect on the storage item.
 */
typedef enum evmc_storage_status (*evmc_set_storage_fn)(struct evmc_context* context,
                                                        const evmc_address* address,
                                                        const evmc_bytes32* key,
                                                        const evmc_bytes32* value);

/**
 * Get balance callback function.
 *
 * This callback function is used by a VM to query the balance of the given account.
 *
 * @param context  The pointer to the Host execution context.
 * @param address  The address of the account.
 * @return         The balance of the given account or 0 if the account does not exist.
 */
typedef evmc_uint256be (*evmc_get_balance_fn)(struct evmc_context* context,
                                              const evmc_address* address);

/**
 * Get code size callback function.
 *
 * This callback function is used by a VM to get the size of the code stored
 * in the account at the given address.
 *
 * @param context  The pointer to the Host execution context.
 * @param address  The address of the account.
 * @return         The size of the code in the account or 0 if the account does not exist.
 */
typedef size_t (*evmc_get_code_size_fn)(struct evmc_context* context, const evmc_address* address);

/**
 * Get code size callback function.
 *
 * This callback function is used by a VM to get the keccak256 hash of the code stored
 * in the account at the given address. For existing accounts not having a code, this
 * function returns keccak256 hash of empty data.
 *
 * @param context  The pointer to the Host execution context.
 * @param address  The address of the account.
 * @return         The hash of the code in the account or null bytes if the account does not exist.
 */
typedef evmc_bytes32 (*evmc_get_code_hash_fn)(struct evmc_context* context,
                                              const evmc_address* address);

/**
 * Copy code callback function.
 *
 *  This callback function is used by an EVM to request a copy of the code
 *  of the given account to the memory buffer provided by the EVM.
 *  The Client MUST copy the requested code, starting with the given offset,
 *  to the provided memory buffer up to the size of the buffer or the size of
 *  the code, whichever is smaller.
 *
 *  @param context      The pointer to the Client execution context.
 *                           @see ::evmc_context.
 *  @param address      The address of the account.
 *  @param code_offset  The offset of the code to copy.
 *  @param buffer_data  The pointer to the memory buffer allocated by the EVM
 *                      to store a copy of the requested code.
 *  @param buffer_size  The size of the memory buffer.
 *  @return             The number of bytes copied to the buffer by the Client.
 */
typedef size_t (*evmc_copy_code_fn)(struct evmc_context* context,
                                    const evmc_address* address,
                                    size_t code_offset,
                                    uint8_t* buffer_data,
                                    size_t buffer_size);

/**
 * Selfdestruct callback function.
 *
 *  This callback function is used by an EVM to SELFDESTRUCT given contract.
 *  The execution of the contract will not be stopped, that is up to the EVM.
 *
 *  @param context      The pointer to the Host execution context.
 *                      @see ::evmc_context.
 *  @param address      The address of the contract to be selfdestructed.
 *  @param beneficiary  The address where the remaining ETH is going to be
 *                      transferred.
 */
typedef void (*evmc_selfdestruct_fn)(struct evmc_context* context,
                                     const evmc_address* address,
                                     const evmc_address* beneficiary);

/**
 * Log callback function.
 *
 *  This callback function is used by an EVM to inform about a LOG that happened
 *  during an EVM bytecode execution.
 *  @param context       The pointer to the Host execution context.
 *                       @see ::evmc_context.
 *  @param address       The address of the contract that generated the log.
 *  @param data          The pointer to unindexed data attached to the log.
 *  @param data_size     The length of the data.
 *  @param topics        The pointer to the array of topics attached to the log.
 *  @param topics_count  The number of the topics. Valid values are between
 *                       0 and 4 inclusively.
 */
typedef void (*evmc_emit_log_fn)(struct evmc_context* context,
                                 const evmc_address* address,
                                 const uint8_t* data,
                                 size_t data_size,
                                 const evmc_bytes32 topics[],
                                 size_t topics_count);

/**
 * Pointer to the callback function supporting EVM calls.
 *
 * @param  context The pointer to the Host execution context.
 * @param  msg     The call parameters.
 * @return         The result of the call.
 */
typedef struct evmc_result (*evmc_call_fn)(struct evmc_context* context,
                                           const struct evmc_message* msg);

/**
 * The Host interface.
 *
 * The set of all callback functions expected by VM instances. This is C
 * realisation of vtable for OOP interface (only virtual methods, no data).
 * Host implementations SHOULD create constant singletons of this (similarly
 * to vtables) to lower the maintenance and memory management cost.
 */
struct evmc_host_interface
{
    /** Check account existence callback function. */
    evmc_account_exists_fn account_exists;

    /** Get storage callback function. */
    evmc_get_storage_fn get_storage;

    /** Set storage callback function. */
    evmc_set_storage_fn set_storage;

    /** Get balance callback function. */
    evmc_get_balance_fn get_balance;

    /** Get code size callback function. */
    evmc_get_code_size_fn get_code_size;

    /** Get code hash callback function. */
    evmc_get_code_hash_fn get_code_hash;

    /** Copy code callback function. */
    evmc_copy_code_fn copy_code;

    /** Selfdestruct callback function. */
    evmc_selfdestruct_fn selfdestruct;

    /** Call callback function. */
    evmc_call_fn call;

    /** Get transaction context callback function. */
    evmc_get_tx_context_fn get_tx_context;

    /** Get block hash callback function. */
    evmc_get_block_hash_fn get_block_hash;

    /** Emit log callback function. */
    evmc_emit_log_fn emit_log;
};


/**
 * Execution context managed by the Host.
 *
 *  The Host MUST pass the pointer to the execution context to
 *  ::evmc_execute_fn. The EVM MUST pass the same pointer back to the Host in
 *  every callback function.
 *  The context MUST contain at least the function table defining the context
 *  callback interface.
 *  Optionally, The Host MAY include in the context additional data.
 */
struct evmc_context
{
    /** The Host interface. */
    const struct evmc_host_interface* host;
};


/* Forward declaration. */
struct evmc_instance;

/**
 * Destroys the EVM instance.
 *
 *  @param evm  The EVM instance to be destroyed.
 */
typedef void (*evmc_destroy_fn)(struct evmc_instance* evm);

/**
 * Possible outcomes of evmc_set_option.
 */
enum evmc_set_option_result
{
    EVMC_SET_OPTION_SUCCESS = 0,
    EVMC_SET_OPTION_INVALID_NAME = 1,
    EVMC_SET_OPTION_INVALID_VALUE = 2
};

/**
 * Configures the EVM instance.
 *
 *  Allows modifying options of the EVM instance.
 *  Options:
 *  - code cache behavior: on, off, read-only, ...
 *  - optimizations,
 *
 *  @param evm    The EVM instance to be configured.
 *  @param name   The option name. NULL-terminated string. Cannot be NULL.
 *  @param value  The new option value. NULL-terminated string. Cannot be NULL.
 *  @return       The outcome of the operation.
 */
typedef enum evmc_set_option_result (*evmc_set_option_fn)(struct evmc_instance* evm,
                                                          char const* name,
                                                          char const* value);


/**
 * EVM revision.
 *
 *  The revision of the EVM specification based on the Ethereum
 *  upgrade / hard fork codenames.
 */
enum evmc_revision
{
    EVMC_FRONTIER = 0,
    EVMC_HOMESTEAD = 1,
    EVMC_TANGERINE_WHISTLE = 2,
    EVMC_SPURIOUS_DRAGON = 3,
    EVMC_BYZANTIUM = 4,
    EVMC_CONSTANTINOPLE = 5,

    EVMC_LATEST_REVISION = EVMC_CONSTANTINOPLE
};


/**
 * Executes the given EVM bytecode using the input in the message
 *
 * This function MAY be invoked multiple times for a single EVM instance.
 *
 * @param instance   The EVM instance.
 * @param context    The pointer to the Client execution context to be passed
 *                   to the callback functions. @see ::evmc_context.
 * @param rev        Requested EVM specification revision.
 * @param msg        Call parameters. @see ::evmc_message.
 * @param code       Reference to the bytecode to be executed.
 * @param code_size  The length of the bytecode.
 * @return           All execution results.
 */
typedef struct evmc_result (*evmc_execute_fn)(struct evmc_instance* instance,
                                              struct evmc_context* context,
                                              enum evmc_revision rev,
                                              const struct evmc_message* msg,
                                              uint8_t const* code,
                                              size_t code_size);

/**
 * Possible capabilities of a VM.
 */
enum evmc_capabilities
{
    EVMC_CAPABILITY_EVM1 = (1u << 0), /**< The VM is capable of executing EVM1 bytecode. */
    EVMC_CAPABILITY_EWASM = (1u << 1) /**< The VM is capable of execution ewasm bytecode. */
};

/**
 * Alias for unsigned integer representing a set of bit flags of EVMC capabilities.
 *
 * @see evmc_capabilities
 */
typedef uint32_t evmc_capabilities_flagset;

/**
 * Return the supported capabilities of the VM instance.
 *
 * This function MAY be invoked multiple times for a single VM instance,
 * and its value MAY be influenced by calls to evmc_instance::set_option.
 *
 * @param instance  The EVM instance.
 * @return          The supported capabilities of the VM. @see evmc_capabilities.
 */
typedef evmc_capabilities_flagset (*evmc_get_capabilities_fn)(struct evmc_instance* instance);

/** The opaque type representing a Client-side tracer object. */
struct evmc_tracer_context;

/**
 * The callback to trace instructions execution in an EVM.
 *
 * This function informs the Client what instruction has been executed in the EVM implementation
 * and what are the results of executing this particular instruction.
 * The message level information (like call depth, destination address, etc.) are not provided here.
 * This piece of information can be acquired by inspecting messages being sent to the EVM in
 * ::evmc_execute_fn and the results of the messages execution.
 *
 * @param context                The pointer to the Client-side tracing context. This allows to
 *                               implement the tracer in OOP manner.
 * @param code_offset            The current instruction position in the code.
 * @param status_code            The status code of the instruction execution.
 * @param gas_left               The amount of the gas left after the instruction execution.
 * @param stack_num_items        The current EVM stack height after the instruction execution.
 * @param pushed_stack_item      The top EVM stack item pushed as the result of the instruction
 *                               execution. This value is null when the instruction does not push
 *                               anything to the stack.
 * @param memory_size            The size of the EVM memory after the instruction execution.
 * @param changed_memory_offset  The offset in number of bytes of the beginning of the memory area
 *                               modified as the result of the instruction execution.
 *                               The Client MAY use this information together with
 *                               @p changed_memory_size and @p changed_memory to incrementally
 *                               update the copy of the full VM's memory.
 * @param changed_memory_size    The size of the memory area modified as the result of
 *                               the instruction execution.
 * @param changed_memory         The pointer to the memory area modified as the result of
 *                               the instruction execution.
 *                               The Client MAY access the pointed memory area
 *                               (limited by the @p changed_memory_size) only during the current
 *                               execution of the evmc_trace_callback().
 *                               The pointer MUST NOT be stored by the Client.
 *                               The Client MUST NOT assume that
 *                               `changed_memory - changed_memory_offset` is a valid base pointer
 *                               of the VM memory.
 */
typedef void (*evmc_trace_callback)(struct evmc_tracer_context* context,
                                    size_t code_offset,
                                    enum evmc_status_code status_code,
                                    int64_t gas_left,
                                    size_t stack_num_items,
                                    const evmc_uint256be* pushed_stack_item,
                                    size_t memory_size,
                                    size_t changed_memory_offset,
                                    size_t changed_memory_size,
                                    const uint8_t* changed_memory);

/**
 * Sets the EVM instruction tracer.
 *
 * When the tracer is set in the EVM instance, the EVM SHOULD call back the tracer with information
 * about instructions execution in the EVM.
 * @see ::evmc_trace_callback.
 *
 * This will overwrite the previous settings (the callback and the context).
 *
 * @param instance    The EVM instance.
 * @param callback    The tracer callback function. This argument MAY be NULL to disable previously
 *                    set tracer.
 * @param context     The Client-side tracer context. This argument MAY be NULL in case the tracer
 *                    does not require any context. This argument MUST be NULL if the callback
 *                    argument is NULL.
 */
typedef void (*evmc_set_tracer_fn)(struct evmc_instance* instance,
                                   evmc_trace_callback callback,
                                   struct evmc_tracer_context* context);


/**
 * The EVM instance.
 *
 *  Defines the base struct of the EVM implementation.
 */
struct evmc_instance
{
    /**
     *  EVMC ABI version implemented by the EVM instance.
     *
     *  Used to detect ABI incompatibilities. The EVMC ABI version
     *  represented by this file is in ::EVMC_ABI_VERSION.
     */
    const int abi_version;

    /**
     * The name of the EVMC VM implementation.
     *
     *  It MUST be a NULL-terminated not empty string.
     */
    const char* name;

    /**
     * The version of the EVMC VM implementation, e.g. "1.2.3b4".
     *
     *  It MUST be a NULL-terminated not empty string.
     */
    const char* version;

    /** Pointer to function destroying the EVM instance. */
    evmc_destroy_fn destroy;

    /** Pointer to function executing a code by the EVM instance. */
    evmc_execute_fn execute;

    /**
     * Pointer to function returning capabilities supported by the VM instance.
     *
     * The value returned might change when different options are requested via set_option.
     *
     * A Client SHOULD only rely on the value returned here if it has queried it after
     * it has called set_option.
     */
    evmc_get_capabilities_fn get_capabilities;

    /**
     * Optional pointer to function setting the EVM instruction tracer.
     *
     * If the EVM does not support this feature the pointer can be NULL.
     */
    evmc_set_tracer_fn set_tracer;

    /**
     * Optional pointer to function modifying VM's options.
     *
     *  If the VM does not support this feature the pointer can be NULL.
     */
    evmc_set_option_fn set_option;
};

/* END Python CFFI declarations */

#if EVMC_DOCUMENTATION
/**
 * Example of a function creating an instance of an example EVM implementation.
 *
 * Each EVM implementation MUST provide a function returning an EVM instance.
 * The function SHOULD be named `evmc_create_<vm-name>(void)`. If the VM name contains hyphens
 * replaces them with underscores in the function names.
 *
 * @par Binaries naming convention
 * For VMs distributed as shared libraries, the name of the library SHOULD match the VM name.
 * The convetional library filename prefixes and extensions SHOULD be ignored by the Client.
 * For example, the shared library with the "beta-interpreter" implementation may be named
 * `libbeta-interpreter.so`.
 *
 * @return  EVM instance or NULL indicating instance creation failure.
 */
struct evmc_instance* evmc_create_example_vm(void);
#endif

#if __cplusplus
}
#endif

#endif
/** @} */
