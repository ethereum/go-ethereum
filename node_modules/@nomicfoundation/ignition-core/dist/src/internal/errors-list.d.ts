import { ErrorDescriptor } from "../types/errors";
export declare const ERROR_PREFIX = "IGN";
export declare function getErrorCode(error: ErrorDescriptor): string;
export declare const ERROR_RANGES: {
    [category in keyof typeof ERRORS]: {
        min: number;
        max: number;
        title: string;
    };
};
/**
 * DEV NOTE:
 *
 * When adding errors, please apply the hardhat-plugin rules
 * and add the new error code to the whitelist if needed.
 */
export declare const ERRORS: {
    GENERAL: {
        ASSERTION_ERROR: {
            number: number;
            message: string;
        };
        UNSUPPORTED_DECODE: {
            number: number;
            message: string;
        };
    };
    INTERNAL: {
        TEMPLATE_INVALID_VARIABLE_NAME: {
            number: number;
            message: string;
        };
        TEMPLATE_VARIABLE_NOT_FOUND: {
            number: number;
            message: string;
        };
        TEMPLATE_VALUE_CONTAINS_VARIABLE_TAG: {
            number: number;
            message: string;
        };
    };
    MODULE: {
        INVALID_MODULE_ID: {
            number: number;
            message: string;
        };
        INVALID_MODULE_ID_CHARACTERS: {
            number: number;
            message: string;
        };
        INVALID_MODULE_DEFINITION_FUNCTION: {
            number: number;
            message: string;
        };
        ASYNC_MODULE_DEFINITION_FUNCTION: {
            number: number;
            message: string;
        };
        DUPLICATE_MODULE_ID: {
            number: number;
            message: string;
        };
    };
    SERIALIZATION: {
        INVALID_FUTURE_ID: {
            number: number;
            message: string;
        };
        INVALID_FUTURE_TYPE: {
            number: number;
            message: string;
        };
        LOOKAHEAD_NOT_FOUND: {
            number: number;
            message: string;
        };
    };
    EXECUTION: {
        DROPPED_TRANSACTION: {
            number: number;
            message: string;
        };
        INVALID_JSON_RPC_RESPONSE: {
            number: number;
            message: string;
        };
        WAITING_FOR_CONFIRMATIONS: {
            number: number;
            message: string;
        };
        WAITING_FOR_NONCE: {
            number: number;
            message: string;
        };
        INVALID_NONCE: {
            number: number;
            message: string;
        };
        BASE_FEE_EXCEEDS_GAS_LIMIT: {
            number: number;
            message: string;
        };
        MAX_FEE_PER_GAS_EXCEEDS_GAS_LIMIT: {
            number: number;
            message: string;
        };
        INSUFFICIENT_FUNDS_FOR_TRANSFER: {
            number: number;
            message: string;
        };
        INSUFFICIENT_FUNDS_FOR_DEPLOY: {
            number: number;
            message: string;
        };
        GAS_ESTIMATION_FAILED: {
            number: number;
            message: string;
        };
    };
    RECONCILIATION: {
        INVALID_EXECUTION_STATUS: {
            number: number;
            message: string;
        };
    };
    WIPE: {
        UNINITIALIZED_DEPLOYMENT: {
            number: number;
            message: string;
        };
        NO_STATE_FOR_FUTURE: {
            number: number;
            message: string;
        };
        DEPENDENT_FUTURES: {
            number: number;
            message: string;
        };
    };
    VALIDATION: {
        INVALID_DEFAULT_SENDER: {
            number: number;
            message: string;
        };
        MISSING_EMITTER: {
            number: number;
            message: string;
        };
        INVALID_MODULE: {
            number: number;
            message: string;
        };
        INVALID_CONSTRUCTOR_ARGS_LENGTH: {
            number: number;
            message: string;
        };
        INVALID_FUNCTION_ARGS_LENGTH: {
            number: number;
            message: string;
        };
        INVALID_STATIC_CALL: {
            number: number;
            message: string;
        };
        INDEXED_EVENT_ARG: {
            number: number;
            message: string;
        };
        INVALID_OVERLOAD_NAME: {
            number: number;
            message: string;
        };
        OVERLOAD_NOT_FOUND: {
            number: number;
            message: string;
        };
        REQUIRE_BARE_NAME: {
            number: number;
            message: string;
        };
        OVERLOAD_NAME_REQUIRED: {
            number: number;
            message: string;
        };
        INVALID_OVERLOAD_GIVEN: {
            number: number;
            message: string;
        };
        EVENT_ARG_NOT_FOUND: {
            number: number;
            message: string;
        };
        INVALID_EVENT_ARG_INDEX: {
            number: number;
            message: string;
        };
        FUNCTION_ARG_NOT_FOUND: {
            number: number;
            message: string;
        };
        INVALID_FUNCTION_ARG_INDEX: {
            number: number;
            message: string;
        };
        MISSING_LIBRARIES: {
            number: number;
            message: string;
        };
        CONFLICTING_LIBRARY_NAMES: {
            number: number;
            message: string;
        };
        INVALID_LIBRARY_NAME: {
            number: number;
            message: string;
        };
        LIBRARY_NOT_NEEDED: {
            number: number;
            message: string;
        };
        AMBIGUOUS_LIBRARY_NAME: {
            number: number;
            message: string;
        };
        INVALID_LIBRARY_ADDRESS: {
            number: number;
            message: string;
        };
        NEGATIVE_ACCOUNT_INDEX: {
            number: number;
            message: string;
        };
        ACCOUNT_INDEX_TOO_HIGH: {
            number: number;
            message: string;
        };
        INVALID_ARTIFACT: {
            number: number;
            message: string;
        };
        MISSING_MODULE_PARAMETER: {
            number: number;
            message: string;
        };
        INVALID_MODULE_PARAMETER_TYPE: {
            number: number;
            message: string;
        };
    };
    STATUS: {
        UNINITIALIZED_DEPLOYMENT: {
            number: number;
            message: string;
        };
    };
    DEPLOY: {
        CHANGED_CHAINID: {
            number: number;
            message: string;
        };
    };
    VERIFY: {
        UNINITIALIZED_DEPLOYMENT: {
            number: number;
            message: string;
        };
        NO_CONTRACTS_DEPLOYED: {
            number: number;
            message: string;
        };
        UNSUPPORTED_CHAIN: {
            number: number;
            message: string;
        };
    };
    STRATEGIES: {
        UNKNOWN_STRATEGY: {
            number: number;
            message: string;
        };
        MISSING_CONFIG: {
            number: number;
            message: string;
        };
        MISSING_CONFIG_PARAM: {
            number: number;
            message: string;
        };
        INVALID_CONFIG_PARAM: {
            number: number;
            message: string;
        };
    };
};
//# sourceMappingURL=errors-list.d.ts.map