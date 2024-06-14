"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ERRORS = exports.ERROR_RANGES = exports.getErrorCode = exports.ERROR_PREFIX = void 0;
exports.ERROR_PREFIX = "IGN";
function getErrorCode(error) {
    return `${exports.ERROR_PREFIX}${error.number}`;
}
exports.getErrorCode = getErrorCode;
exports.ERROR_RANGES = {
    GENERAL: {
        min: 1,
        max: 99,
        title: "General errors",
    },
    INTERNAL: {
        min: 100,
        max: 199,
        title: "Internal Hardhat Ignition errors",
    },
    MODULE: {
        min: 200,
        max: 299,
        title: "Module related errors",
    },
    SERIALIZATION: {
        min: 300,
        max: 399,
        title: "Serialization errors",
    },
    EXECUTION: {
        min: 400,
        max: 499,
        title: "Execution errors",
    },
    RECONCILIATION: {
        min: 500,
        max: 599,
        title: "Reconciliation errors",
    },
    WIPE: {
        min: 600,
        max: 699,
        title: "Wipe errors",
    },
    VALIDATION: {
        min: 700,
        max: 799,
        title: "Validation errors",
    },
    STATUS: {
        min: 800,
        max: 899,
        title: "Status errors",
    },
    DEPLOY: {
        min: 900,
        max: 999,
        title: "Deploy errors",
    },
    VERIFY: {
        min: 1000,
        max: 1099,
        title: "Verify errors",
    },
    STRATEGIES: {
        min: 1100,
        max: 1199,
        title: "Strategy errors",
    },
};
/**
 * DEV NOTE:
 *
 * When adding errors, please apply the hardhat-plugin rules
 * and add the new error code to the whitelist if needed.
 */
exports.ERRORS = {
    GENERAL: {
        ASSERTION_ERROR: {
            number: 1,
            message: "Internal Hardhat Ignition invariant was violated: %description%",
        },
        UNSUPPORTED_DECODE: {
            number: 2,
            message: "Hardhat Ignition can't decode ethers.js value of type %type%: %value%",
        },
    },
    INTERNAL: {
        TEMPLATE_INVALID_VARIABLE_NAME: {
            number: 100,
            message: "Variable names can only include ascii letters and numbers, and start with a letter, but got %variable%",
        },
        TEMPLATE_VARIABLE_NOT_FOUND: {
            number: 101,
            message: "Variable %variable%'s tag not present in the template",
        },
        TEMPLATE_VALUE_CONTAINS_VARIABLE_TAG: {
            number: 102,
            message: "Template values can't include variable tags, but %variable%'s value includes one",
        },
    },
    MODULE: {
        INVALID_MODULE_ID: {
            number: 200,
            message: "Module id must be a string",
        },
        INVALID_MODULE_ID_CHARACTERS: {
            number: 201,
            message: 'The moduleId "%moduleId%" is invalid. Module ids can only have alphanumerics and underscore, and they must start with an alphanumeric.',
        },
        INVALID_MODULE_DEFINITION_FUNCTION: {
            number: 202,
            message: "Module definition function must be a function.",
        },
        ASYNC_MODULE_DEFINITION_FUNCTION: {
            number: 203,
            message: "The callback passed to 'buildModule' for %moduleDefinitionId% returns a Promise; async callbacks are not allowed in 'buildModule'.",
        },
        DUPLICATE_MODULE_ID: {
            number: 204,
            message: "The following module ids are duplicated: %duplicateModuleIds%. Please make sure all module ids are unique.",
        },
    },
    SERIALIZATION: {
        INVALID_FUTURE_ID: {
            number: 300,
            message: "Unable to lookup future during deserialization: %futureId%",
        },
        INVALID_FUTURE_TYPE: {
            number: 301,
            message: "Invalid FutureType %type% as serialized argument",
        },
        LOOKAHEAD_NOT_FOUND: {
            number: 302,
            message: "Lookahead value %key% missing",
        },
    },
    EXECUTION: {
        DROPPED_TRANSACTION: {
            number: 401,
            message: "Error while executing %futureId%: all the transactions of its network interaction %networkInteractionId% were dropped. Please try rerunning Hardhat Ignition.",
        },
        INVALID_JSON_RPC_RESPONSE: {
            number: 402,
            message: "Invalid JSON-RPC response for %method%: %response%",
        },
        WAITING_FOR_CONFIRMATIONS: {
            number: 403,
            message: "You have sent transactions from %sender% and they interfere with Hardhat Ignition. Please wait until they get %requiredConfirmations% confirmations before running Hardhat Ignition again.",
        },
        WAITING_FOR_NONCE: {
            number: 404,
            message: "You have sent transactions from %sender% with nonce %nonce% and it interferes with Hardhat Ignition. Please wait until they get %requiredConfirmations% confirmations before running Hardhat Ignition again.",
        },
        INVALID_NONCE: {
            number: 405,
            message: "The next nonce for %sender% should be %expectedNonce%, but is %pendingCount%. Please make sure not to send transactions from %sender% while running this deployment and try again.",
        },
        BASE_FEE_EXCEEDS_GAS_LIMIT: {
            number: 406,
            message: "The configured base fee exceeds the block gas limit. Please reduce the configured base fee or increase the block gas limit.",
        },
        MAX_FEE_PER_GAS_EXCEEDS_GAS_LIMIT: {
            number: 407,
            message: "The calculated max fee per gas exceeds the configured limit.",
        },
        INSUFFICIENT_FUNDS_FOR_TRANSFER: {
            number: 408,
            message: "Account %sender% has insufficient funds to transfer %amount% wei",
        },
        INSUFFICIENT_FUNDS_FOR_DEPLOY: {
            number: 409,
            message: "Account %sender% has insufficient funds to deploy the contract",
        },
        GAS_ESTIMATION_FAILED: {
            number: 410,
            message: "Gas estimation failed: %error%",
        },
    },
    RECONCILIATION: {
        INVALID_EXECUTION_STATUS: {
            number: 500,
            message: "Unsupported execution status: %status%",
        },
    },
    WIPE: {
        UNINITIALIZED_DEPLOYMENT: {
            number: 600,
            message: "Cannot wipe %futureId% as the deployment hasn't been intialialized yet",
        },
        NO_STATE_FOR_FUTURE: {
            number: 601,
            message: "Cannot wipe %futureId% as it has no previous execution recorded",
        },
        DEPENDENT_FUTURES: {
            number: 602,
            message: `Cannot wipe %futureId% as there are dependent futures that have previous executions recorded. Consider wiping these first: %dependents%`,
        },
    },
    VALIDATION: {
        INVALID_DEFAULT_SENDER: {
            number: 700,
            message: "Default sender %defaultSender% is not part of the configured accounts.",
        },
        MISSING_EMITTER: {
            number: 701,
            message: "`options.emitter` must be provided when reading an event from a SendDataFuture",
        },
        INVALID_MODULE: {
            number: 702,
            message: "Module validation failed with reason: %message%",
        },
        INVALID_CONSTRUCTOR_ARGS_LENGTH: {
            number: 703,
            message: "The constructor of the contract '%contractName%' expects %expectedArgsLength% arguments but %argsLength% were given",
        },
        INVALID_FUNCTION_ARGS_LENGTH: {
            number: 704,
            message: "Function %functionName% in contract %contractName% expects %expectedLength% arguments but %argsLength% were given",
        },
        INVALID_STATIC_CALL: {
            number: 705,
            message: "Function %functionName% in contract %contractName% is not 'pure' or 'view' and should not be statically called",
        },
        INDEXED_EVENT_ARG: {
            number: 706,
            message: "Indexed argument %argument% of event %eventName% of contract %contractName% is not stored in the receipt (its hash is stored instead), so you can't read it.",
        },
        INVALID_OVERLOAD_NAME: {
            number: 707,
            message: "Invalid %eventOrFunction% name '%name%'",
        },
        OVERLOAD_NOT_FOUND: {
            number: 708,
            message: "%eventOrFunction% '%name%' not found in contract %contractName%",
        },
        REQUIRE_BARE_NAME: {
            number: 709,
            message: "%eventOrFunction% name '%name%' used for contract %contractName%, but it's not overloaded. Use '%bareName%' instead.",
        },
        OVERLOAD_NAME_REQUIRED: {
            number: 710,
            message: `%eventOrFunction% '%name%' is overloaded in contract %contractName%. Please use one of these names instead:

%normalizedNameList%`,
        },
        INVALID_OVERLOAD_GIVEN: {
            number: 711,
            message: `%eventOrFunction% '%name%' is not a valid overload of '%bareName%' in contract %contractName%. Please use one of these names instead:

%normalizedNameList%`,
        },
        EVENT_ARG_NOT_FOUND: {
            number: 712,
            message: "Event %eventName% of contract %contractName% has no argument named %argument%",
        },
        INVALID_EVENT_ARG_INDEX: {
            number: 713,
            message: "Event %eventName% of contract %contractName% has only %expectedLength% arguments, but argument %argument% was requested",
        },
        FUNCTION_ARG_NOT_FOUND: {
            number: 714,
            message: "Function %functionName% of contract %contractName% has no return value named %argument%",
        },
        INVALID_FUNCTION_ARG_INDEX: {
            number: 715,
            message: "Function %functionName% of contract %contractName% has only %expectedLength% return values, but value %argument% was requested",
        },
        MISSING_LIBRARIES: {
            number: 716,
            message: "Invalid libraries for contract %contractName%: The following libraries are missing: %fullyQualifiedNames%",
        },
        CONFLICTING_LIBRARY_NAMES: {
            number: 717,
            message: "Invalid libraries for contract %contractName%: The names '%inputName%' and '%libName%' clash with each other, please use qualified names for both.",
        },
        INVALID_LIBRARY_NAME: {
            number: 718,
            message: "Invalid library name %libraryName% for contract %contractName%",
        },
        LIBRARY_NOT_NEEDED: {
            number: 719,
            message: "Invalid library %libraryName% for contract %contractName%: this library is not needed by this contract.",
        },
        AMBIGUOUS_LIBRARY_NAME: {
            number: 720,
            message: `Invalid libraries for contract %contractName%: The name "%libraryName%" is ambiguous, please use one of the following fully qualified names: %fullyQualifiedNames%`,
        },
        INVALID_LIBRARY_ADDRESS: {
            number: 721,
            message: `Invalid address %address% for library %libraryName% of contract %contractName%`,
        },
        NEGATIVE_ACCOUNT_INDEX: {
            number: 722,
            message: "Account index cannot be a negative number",
        },
        ACCOUNT_INDEX_TOO_HIGH: {
            number: 723,
            message: "Requested account index '%accountIndex%' is greater than the total number of available accounts '%accountsLength%'",
        },
        INVALID_ARTIFACT: {
            number: 724,
            message: "Artifact for contract '%contractName%' is invalid",
        },
        MISSING_MODULE_PARAMETER: {
            number: 725,
            message: "Module parameter '%name%' requires a value but was given none",
        },
        INVALID_MODULE_PARAMETER_TYPE: {
            number: 726,
            message: `Module parameter '%name%' must be of type '%expectedType%' but is '%actualType%'`,
        },
    },
    STATUS: {
        UNINITIALIZED_DEPLOYMENT: {
            number: 800,
            message: "Cannot get status for nonexistant deployment at %deploymentDir%",
        },
    },
    DEPLOY: {
        CHANGED_CHAINID: {
            number: 900,
            message: `The deployment's chain cannot be changed between runs. The deployment was previously run against the chain %previousChainId%, but the current network is the chain %currentChainId%.`,
        },
    },
    VERIFY: {
        UNINITIALIZED_DEPLOYMENT: {
            number: 1000,
            message: "Cannot verify contracts for nonexistant deployment at %deploymentDir%",
        },
        NO_CONTRACTS_DEPLOYED: {
            number: 1001,
            message: "Cannot verify deployment %deploymentDir% as no contracts were deployed",
        },
        UNSUPPORTED_CHAIN: {
            number: 1002,
            message: "Verification not natively supported for chainId %chainId%. Please use a custom chain configuration to add support.",
        },
    },
    STRATEGIES: {
        UNKNOWN_STRATEGY: {
            number: 1100,
            message: "Unknown strategy %strategyName%, must be either 'basic' or 'create2'",
        },
        MISSING_CONFIG: {
            number: 1101,
            message: "No strategy config passed for strategy '%strategyName%'",
        },
        MISSING_CONFIG_PARAM: {
            number: 1102,
            message: "Missing required strategy configuration parameter '%requiredParam%' for the strategy '%strategyName%'",
        },
        INVALID_CONFIG_PARAM: {
            number: 1102,
            message: "Strategy configuration parameter '%paramName%' for the strategy '%strategyName%' is invalid: %reason%",
        },
    },
};
/**
 * Setting the type of ERRORS to a map let us access undefined ones. Letting it
 * be a literal doesn't enforce that its values are of type ErrorDescriptor.
 *
 * We let it be a literal, and use this variable to enforce the types
 */
const _PHONY_VARIABLE_TO_FORCE_ERRORS_TO_BE_OF_TYPE_ERROR_DESCRIPTOR = exports.ERRORS;
//# sourceMappingURL=errors-list.js.map