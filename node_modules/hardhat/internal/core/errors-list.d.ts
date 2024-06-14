export declare const ERROR_PREFIX = "HH";
export interface ErrorDescriptor {
    number: number;
    message: string;
    title: string;
    description: string;
    shouldBeReported: boolean;
}
export declare function getErrorCode(error: ErrorDescriptor): string;
export declare const ERROR_RANGES: {
    [category in keyof typeof ERRORS]: {
        min: number;
        max: number;
        title: string;
    };
};
export declare const ERRORS: {
    GENERAL: {
        NOT_INSIDE_PROJECT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_NODE_VERSION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNSUPPORTED_OPERATION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CONTEXT_ALREADY_CREATED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CONTEXT_NOT_CREATED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CONTEXT_HRE_NOT_DEFINED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CONTEXT_HRE_ALREADY_DEFINED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_CONFIG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        LIB_IMPORTED_FROM_THE_CONFIG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        USER_CONFIG_MODIFIED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ASSERTION_ERROR: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NON_LOCAL_INSTALLATION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        TS_NODE_NOT_INSTALLED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        TYPESCRIPT_NOT_INSTALLED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NOT_INSIDE_PROJECT_ON_WINDOWS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CONFLICTING_FILES: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_BIG_NUMBER: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CORRUPTED_LOCKFILE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ESM_PROJECT_WITHOUT_CJS_CONFIG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ESM_TYPESCRIPT_PROJECT_CREATION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNINITIALIZED_PROVIDER: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_READ_OF_DIRECTORY: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        HARDHAT_PROJECT_ALREADY_CREATED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NOT_IN_INTERACTIVE_SHELL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    NETWORK: {
        CONFIG_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_GLOBAL_CHAIN_ID: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ETHSIGN_MISSING_DATA_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NOT_LOCAL_ACCOUNT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MISSING_TX_PARAM_TO_SIGN_LOCALLY: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NO_REMOTE_ACCOUNT_AVAILABLE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_HD_PATH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_RPC_QUANTITY_VALUE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NODE_IS_NOT_RUNNING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NETWORK_TIMEOUT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_JSON_RESPONSE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CANT_DERIVE_KEY: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_RPC_DATA_VALUE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ETHSIGN_TYPED_DATA_V4_INVALID_DATA_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INCOMPATIBLE_FEE_PRICE_FIELDS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MISSING_FEE_PRICE_FIELDS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        PERSONALSIGN_MISSING_ADDRESS_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        EMPTY_URL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    TASK_DEFINITIONS: {
        PARAM_AFTER_VARIADIC: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        PARAM_ALREADY_DEFINED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        PARAM_CLASHES_WITH_HARDHAT_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MANDATORY_PARAM_AFTER_OPTIONAL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ACTION_NOT_SET: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        RUNSUPER_NOT_AVAILABLE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        DEFAULT_VALUE_WRONG_TYPE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        DEFAULT_IN_MANDATORY_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_PARAM_NAME_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        OVERRIDE_NO_MANDATORY_PARAMS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        OVERRIDE_NO_POSITIONAL_PARAMS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        OVERRIDE_NO_VARIADIC_PARAMS: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CLI_ARGUMENT_TYPE_REQUIRED: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        TASK_SCOPE_CLASH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        SCOPE_TASK_CLASH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        DEPRECATED_TRANSFORM_IMPORT_TASK: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    ARGUMENTS: {
        INVALID_ENV_VAR_VALUE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_VALUE_FOR_TYPE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_INPUT_FILE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_TASK: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_COMMAND_LINE_ARG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_PARAM_NAME: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MISSING_TASK_ARGUMENT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MISSING_POSITIONAL_ARG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_POSITIONAL_ARG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        REPEATED_PARAM: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        PARAM_NAME_INVALID_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_JSON_ARGUMENT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        RUNNING_SUBTASK_FROM_CLI: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        TYPECHECK_USED_IN_JAVASCRIPT_PROJECT: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_SCOPE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        UNRECOGNIZED_SCOPED_TASK: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    RESOLVER: {
        FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        LIBRARY_NOT_INSTALLED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        LIBRARY_FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        ILLEGAL_IMPORT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        IMPORTED_FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_BACKSLASH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_PROTOCOL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_ABSOLUTE_PATH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_OUTSIDE_OF_PROJECT: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_WRONG_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        WRONG_SOURCE_NAME_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        IMPORTED_LIBRARY_NOT_INSTALLED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INCLUDES_OWN_PACKAGE_NAME: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        IMPORTED_MAPPED_FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_IMPORT_OF_DIRECTORY: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        AMBIGUOUS_SOURCE_NAMES: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    SOLC: {
        INVALID_VERSION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        DOWNLOAD_FAILED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        VERSION_LIST_DOWNLOAD_FAILED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_DOWNLOAD: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CANT_GET_COMPILER: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        CANT_RUN_NATIVE_COMPILER: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        SOLCJS_ERROR: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    BUILTIN_TASKS: {
        COMPILE_FAILURE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        RUN_FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        RUN_SCRIPT_ERROR: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        FLATTEN_CYCLE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        JSONRPC_SERVER_ERROR: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        JSONRPC_UNSUPPORTED_NETWORK: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        COMPILATION_JOBS_CREATION_FAILURE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NODE_FORK_BLOCK_NUMBER_WITHOUT_URL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        COMPILE_TASK_UNSUPPORTED_SOLC_VERSION: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        TEST_TASK_ESM_TESTS_RUN_TWICE: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    ARTIFACTS: {
        NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MULTIPLE_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        WRONG_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    PLUGINS: {
        BUIDLER_PLUGIN: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        MISSING_DEPENDENCIES: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    INTERNAL: {
        TEMPLATE_INVALID_VARIABLE_NAME: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        TEMPLATE_VALUE_CONTAINS_VARIABLE_TAG: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        TEMPLATE_VARIABLE_TAG_MISSING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        WRONG_ARTIFACT_PATH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    SOURCE_NAMES: {
        INVALID_SOURCE_NAME_ABSOLUTE_PATH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_SOURCE_NAME_RELATIVE_PATH: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_SOURCE_NAME_BACKSLASHES: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_SOURCE_NOT_NORMALIZED: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        WRONG_CASING: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        FILE_NOT_FOUND: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        NODE_MODULES_AS_LOCAL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
        EXTERNAL_AS_LOCAL: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    CONTRACT_NAMES: {
        INVALID_FULLY_QUALIFIED_NAME: {
            number: number;
            message: string;
            title: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
    VARS: {
        ONLY_MANAGED_IN_CLI: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        VALUE_NOT_FOUND_FOR_VAR: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_CONFIG_VAR_NAME: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
        INVALID_EMPTY_VALUE: {
            number: number;
            title: string;
            message: string;
            description: string;
            shouldBeReported: boolean;
        };
    };
};
//# sourceMappingURL=errors-list.d.ts.map