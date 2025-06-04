"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ERRORS = exports.ERROR_RANGES = exports.getErrorCode = exports.ERROR_PREFIX = void 0;
exports.ERROR_PREFIX = "HH";
function getErrorCode(error) {
    return `${exports.ERROR_PREFIX}${error.number}`;
}
exports.getErrorCode = getErrorCode;
exports.ERROR_RANGES = {
    GENERAL: { min: 1, max: 99, title: "General errors" },
    NETWORK: { min: 100, max: 199, title: "Network related errors" },
    TASK_DEFINITIONS: {
        min: 200,
        max: 299,
        title: "Task definition errors",
    },
    ARGUMENTS: { min: 300, max: 399, title: "Arguments related errors" },
    RESOLVER: {
        min: 400,
        max: 499,
        title: "Dependencies resolution errors",
    },
    SOLC: { min: 500, max: 599, title: "Solidity related errors" },
    BUILTIN_TASKS: { min: 600, max: 699, title: "Built-in tasks errors" },
    ARTIFACTS: { min: 700, max: 799, title: "Artifacts related errors" },
    PLUGINS: { min: 800, max: 899, title: "Plugin system errors" },
    INTERNAL: { min: 900, max: 999, title: "Internal Hardhat errors" },
    SOURCE_NAMES: { min: 1000, max: 1099, title: "Source name errors" },
    CONTRACT_NAMES: { min: 1100, max: 1199, title: "Contract name errors" },
    VARS: { min: 1200, max: 1299, title: "Connfiguration variables errors" },
};
exports.ERRORS = {
    GENERAL: {
        NOT_INSIDE_PROJECT: {
            number: 1,
            message: "You are not inside a Hardhat project.",
            title: "You are not inside a Hardhat project",
            description: `You are trying to run Hardhat outside of a Hardhat project.

You can learn how to use Hardhat by reading the [Getting Started guide](/hardhat-runner/docs/getting-started).`,
            shouldBeReported: false,
        },
        INVALID_NODE_VERSION: {
            number: 2,
            message: "Hardhat doesn't support your Node.js version. It should be %requirement%.",
            title: "Unsupported Node.js",
            description: `Hardhat doesn't support your Node.js version.

Please upgrade your version of Node.js and try again.`,
            shouldBeReported: false,
        },
        UNSUPPORTED_OPERATION: {
            number: 3,
            message: "%operation% is not supported in Hardhat.",
            title: "Unsupported operation",
            description: `You are trying to perform an unsupported operation.

Unless you are creating a task or plugin, this is probably a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        CONTEXT_ALREADY_CREATED: {
            number: 4,
            message: "HardhatContext is already created.",
            title: "Hardhat was already initialized",
            description: `Hardhat initialization was executed twice. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        CONTEXT_NOT_CREATED: {
            number: 5,
            message: "HardhatContext is not created.",
            title: "Hardhat wasn't initialized",
            description: `Hardhat initialization failed. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        CONTEXT_HRE_NOT_DEFINED: {
            number: 6,
            message: "Hardhat Runtime Environment is not defined in the HardhatContext.",
            title: "Hardhat Runtime Environment not created",
            description: `Hardhat initialization failed. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        CONTEXT_HRE_ALREADY_DEFINED: {
            number: 7,
            message: "Hardhat Runtime Environment is already defined in the HardhatContext",
            title: "Tried to create the Hardhat Runtime Environment twice",
            description: `The Hardhat initialization process was executed twice. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        INVALID_CONFIG: {
            number: 8,
            message: `There's one or more errors in your config file:

%errors%

To learn more about Hardhat's configuration, please go to https://hardhat.org/config/`,
            title: "Invalid Hardhat config",
            description: `You have one or more errors in your config file.

Check the error message for details, or go to the [documentation](https://hardhat.org/config/) to learn more.`,
            shouldBeReported: false,
        },
        LIB_IMPORTED_FROM_THE_CONFIG: {
            number: 9,
            message: `Error while loading Hardhat's configuration.

You probably tried to import the "hardhat" module from your config or a file imported from it.
This is not possible, as Hardhat can't be initialized while its config is being defined.

To learn more about how to access the Hardhat Runtime Environment from different contexts go to https://hardhat.org/hre`,
            title: "Failed to load config file",
            description: `There was an error while loading your config file.

The most common source of errors is trying to import the Hardhat Runtime Environment from your config or a file imported from it.
This is not possible, as Hardhat can't be initialized while its config is being defined.

You may also have accidentally imported \`hardhat\` instead of \`hardhat/config\`.

Please make sure your config file is correct.

To learn more about how to access the Hardhat Runtime Environment from different contexts go to https://hardhat.org/hre`,
            shouldBeReported: false,
        },
        USER_CONFIG_MODIFIED: {
            number: 10,
            message: `Error while loading Hardhat's configuration.
You or one of your plugins is trying to modify the userConfig.%path% value from a config extender`,
            title: "Attempted to modify the user's config",
            description: `An attempt to modify the user's config was made.

This is probably a bug in one of your plugins.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        ASSERTION_ERROR: {
            number: 11,
            message: "An internal invariant was violated: %message%",
            title: "Invariant violation",
            description: `An internal invariant was violated.
This is probably caused by a programming error in hardhat or in one of the used plugins.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        NON_LOCAL_INSTALLATION: {
            number: 12,
            message: "Trying to use a non-local installation of Hardhat, which is not supported.\nPlease install Hardhat locally using npm or Yarn, and try again.",
            title: "Hardhat is not installed or installed globally",
            description: `You tried to run Hardhat from a global installation or not installing it at all. This is not supported.

Please install Hardhat locally using npm or Yarn, and try again.`,
            shouldBeReported: false,
        },
        TS_NODE_NOT_INSTALLED: {
            number: 13,
            message: `Your Hardhat project uses typescript, but ts-node is not installed.

Please run: npm install --save-dev ts-node`,
            title: "ts-node not installed",
            description: `You are running a Hardhat project that uses typescript, but you haven't installed ts-node.

Please run this and try again: \`npm install --save-dev ts-node\``,
            shouldBeReported: false,
        },
        TYPESCRIPT_NOT_INSTALLED: {
            number: 14,
            message: `Your Hardhat project uses typescript, but it's not installed.

Please run: npm install --save-dev typescript`,
            title: "typescript not installed",
            description: `You are running a Hardhat project that uses typescript, but it's not installed.

Please run this and try again: \`npm install --save-dev typescript\``,
            shouldBeReported: false,
        },
        NOT_INSIDE_PROJECT_ON_WINDOWS: {
            number: 15,
            message: `You are not inside a project and Hardhat failed to initialize a new one.

If you were trying to create a new project, please try again using Windows Subsystem for Linux (WSL) or PowerShell.
`,
            title: "You are not inside a Hardhat project and Hardhat failed to initialize a new one",
            description: `You are trying to run Hardhat outside of a Hardhat project, and we couldn't initialize one.

If you were trying to create a new project, please try again using Windows Subsystem for Linux (WSL) or PowerShell.

You can learn how to use Hardhat by reading the [Getting Started guide](/hardhat-runner/docs/getting-started).`,
            shouldBeReported: false,
        },
        CONFLICTING_FILES: {
            number: 16,
            message: `The directory %dest% contains files that could conflict:

%conflicts%

Either try using a new directory, or remove the files listed above.`,
            title: "conflicting files during project creation",
            description: `You are trying to create a new hardhat project, but there are existing files that would be overwritten by the creation process.

Either try using a new directory name, or remove the conflicting files.`,
            shouldBeReported: false,
        },
        INVALID_BIG_NUMBER: {
            number: 17,
            message: "The input value cannot be normalized to a BigInt: %message%",
            title: "Invalid big number",
            description: "Hardhat attempted to convert the input value to a BigInt, but no known conversion method was applicable to the given value.",
            shouldBeReported: false,
        },
        CORRUPTED_LOCKFILE: {
            number: 18,
            message: `You installed Hardhat with a corrupted lockfile due to the NPM bug #4828.

Please delete your node_modules, package-lock.json, reinstall your project, and try again.`,
            title: "Corrupted lockfile",
            description: `Some versions of NPM are affected [by a bug](https://github.com/npm/cli/issues/4828) that leads to corrupt lockfiles being generated.

This bug can only affect you if you, or someone at your team, installed the project without a lockfile, but with an existing node_modules.

To avoid it, please delete both your node_modules and package-lock.json, and reinstall your project.

Note that you don't need to do this every time you install a new dependency, but please make sure to delete your node_modules every time you delete your package-lock.json.`,
            shouldBeReported: true,
        },
        ESM_PROJECT_WITHOUT_CJS_CONFIG: {
            number: 19,
            message: `Your project is an ESM project (you have "type": "module" set in your package.json) but your Hardhat config file uses the .js extension.

Rename the file to use the .cjs to fix this problem.`,
            title: "Hardhat config with .js extension in an ESM project",
            description: "Your project is an ESM project but your Hardhat config uses the .js extension. Hardhat config files cannot be an ES module. To fix this, rename your Hardhat config to use the .cjs extension.",
            shouldBeReported: false,
        },
        ESM_TYPESCRIPT_PROJECT_CREATION: {
            number: 20,
            message: `Your project is an ESM project (you have "type": "module" set in your package.json) and you are trying to initialize a TypeScript project. This is not supported yet.`,
            title: "Initializing a TypeScript sample project in an ESM project",
            description: `Your project is an ESM project (you have "type": "module" set in your package.json) and you are trying to initialize a TypeScript project. This is not supported yet.`,
            shouldBeReported: false,
        },
        UNINITIALIZED_PROVIDER: {
            number: 21,
            message: "You tried to access an uninitialized provider. To initialize the provider, make sure you first call `.init()` or any method that hits a node like request, send or sendAsync.",
            title: "Uninitialized provider",
            description: `You tried to access an uninitialized provider. This is most likely caused by using the internal wrapped provider directly before using it to send a request or initializing it.
To initialize the provider, make sure you first call \`.init()\` or any method that hits a node like request, send or sendAsync.`,
            shouldBeReported: true,
        },
        INVALID_READ_OF_DIRECTORY: {
            number: 22,
            message: "Invalid file path %absolutePath%. Attempting to read a directory instead of a file.",
            title: "Invalid read: a directory cannot be read",
            description: `An attempt was made to read a file, but a path to a directory was provided.

Please double check the file path.`,
            shouldBeReported: false,
        },
        HARDHAT_PROJECT_ALREADY_CREATED: {
            number: 23,
            message: "You are trying to initialize a project inside an existing Hardhat project. The path to the project's configuration file is:  %hardhatProjectRootPath%.",
            title: "Hardhat project already created",
            description: `Cannot create a new Hardhat project, the current folder is already associated with a project.`,
            shouldBeReported: false,
        },
        NOT_IN_INTERACTIVE_SHELL: {
            number: 24,
            message: "You are trying to initialize a project but you are not in an interactive shell.",
            title: "Not inside an interactive shell",
            description: `You are trying to initialize a project but you are not in an interactive shell.

Please re-run the command inside an interactive shell.`,
            shouldBeReported: false,
        },
    },
    NETWORK: {
        CONFIG_NOT_FOUND: {
            number: 100,
            message: "Network %network% doesn't exist",
            title: "Selected network doesn't exist",
            description: `You are trying to run Hardhat with a nonexistent network.

Read the [documentation](https://hardhat.org/hardhat-runner/docs/config#networks-configuration) to learn how to define custom networks.`,
            shouldBeReported: false,
        },
        INVALID_GLOBAL_CHAIN_ID: {
            number: 101,
            message: "Hardhat was set to use chain id %configChainId%, but connected to a chain with id %connectionChainId%.",
            title: "Connected to the wrong network",
            description: `Your config specifies a chain id for the network you are trying to use, but Hardhat detected a different chain id.

Please make sure you are setting your config correctly.`,
            shouldBeReported: false,
        },
        ETHSIGN_MISSING_DATA_PARAM: {
            number: 102,
            message: 'Missing "data" param when calling eth_sign.',
            title: "Missing `data` param when calling eth_sign.",
            description: `You called \`eth_sign\` with incorrect parameters.

Please check that you are sending a \`data\` parameter.`,
            shouldBeReported: false,
        },
        NOT_LOCAL_ACCOUNT: {
            number: 103,
            message: "Account %account% is not managed by the node you are connected to.",
            title: "Unrecognized account",
            description: `You are trying to send a transaction or sign some data with an
account not managed by your Ethereum node nor Hardhat.

Please double check your accounts and the \`from\` parameter in your RPC calls.`,
            shouldBeReported: false,
        },
        MISSING_TX_PARAM_TO_SIGN_LOCALLY: {
            number: 104,
            message: "Missing param %param% from a tx being signed locally.",
            title: "Missing transaction parameter",
            description: `You are trying to send a transaction with a locally managed
account, and some parameters are missing.

Please double check your transactions' parameters.`,
            shouldBeReported: false,
        },
        NO_REMOTE_ACCOUNT_AVAILABLE: {
            number: 105,
            message: "No local account was set and there are accounts in the remote node.",
            title: "No remote accounts available",
            description: `No local account was set and there are accounts in the remote node.

Please make sure that your Ethereum node has unlocked accounts.`,
            shouldBeReported: false,
        },
        INVALID_HD_PATH: {
            number: 106,
            message: "HD path %path% is invalid. Read about BIP32 to know about the valid forms.",
            title: "Invalid HD path",
            description: `An invalid HD/BIP32 derivation path was provided in your config.

Read the [documentation](https://hardhat.org/hardhat-runner/docs/config#hd-wallet-config) to learn how to define HD accounts correctly.`,
            shouldBeReported: false,
        },
        INVALID_RPC_QUANTITY_VALUE: {
            number: 107,
            message: "Received invalid value `%value%` from/to the node's JSON-RPC, but a Quantity was expected.",
            title: "Invalid JSON-RPC value",
            description: `One of your transactions sent or received an invalid JSON-RPC QUANTITY value.

Please double check your calls' parameters and keep your Ethereum node up to date.`,
            shouldBeReported: false,
        },
        NODE_IS_NOT_RUNNING: {
            number: 108,
            message: `Cannot connect to the network %network%.
Please make sure your node is running, and check your internet connection and networks config`,
            title: "Cannot connect to the network",
            description: `Cannot connect to the network.

Please make sure your node is running, and check your internet connection and networks config.`,
            shouldBeReported: false,
        },
        NETWORK_TIMEOUT: {
            number: 109,
            message: `Network connection timed out.
Please check your internet connection and networks config`,
            title: "Network timeout",
            description: `One of your JSON-RPC requests timed out.

Please make sure your node is running, and check your internet connection and networks config.`,
            shouldBeReported: false,
        },
        INVALID_JSON_RESPONSE: {
            number: 110,
            message: "Invalid JSON-RPC response received: %response%",
            title: "Invalid JSON-RPC response",
            description: `One of your JSON-RPC requests received an invalid response.

Please make sure your node is running, and check your internet connection and networks config.`,
            shouldBeReported: false,
        },
        CANT_DERIVE_KEY: {
            number: 111,
            message: "Cannot derive key %path% from mnemonic '%mnemonic%.\nTry using another mnemonic or deriving fewer keys.",
            title: "Could not derive an HD key",
            description: `One of your HD keys could not be derived.

Try using another mnemonic or deriving less keys.`,
            shouldBeReported: false,
        },
        INVALID_RPC_DATA_VALUE: {
            number: 112,
            message: "Received invalid value `%value%` from/to the node's JSON-RPC, but a Data was expected.",
            title: "Invalid JSON-RPC value",
            description: `One of your calls sent or received an invalid JSON-RPC DATA value.

Please double check your calls' parameters and keep your Ethereum node up to date.`,
            shouldBeReported: false,
        },
        ETHSIGN_TYPED_DATA_V4_INVALID_DATA_PARAM: {
            number: 113,
            message: 'Invalid "data" param when calling eth_signTypedData_v4.',
            title: "Invalid `data` param when calling eth_signTypedData_v4.",
            description: `You called \`eth_signTypedData_v4\` with incorrect parameters.
Please check that you are sending a \`data\` parameter with a JSON string or object conforming to EIP712 TypedData schema.`,
            shouldBeReported: false,
        },
        INCOMPATIBLE_FEE_PRICE_FIELDS: {
            number: 114,
            message: "An incompatible transaction with gasPrice and EIP-1559 fee price fields.",
            title: "Incompatible fee price parameters",
            description: `You are trying to send a transaction with a locally managed
account, and its parameters are incompatible. You sent both gasPrice, and maxFeePerGas or maxPriorityFeePerGas.

Please double check your transactions' parameters.`,
            shouldBeReported: false,
        },
        MISSING_FEE_PRICE_FIELDS: {
            number: 115,
            message: "Tried to sign a transaction locally, but gasPrice, maxFeePerGas, and maxPriorityFeePerGas were missing.",
            title: "Missing fee price parameters",
            description: `You are trying to send a transaction with a locally managed account, and no fee price parameters were provided. You need to send gasPrice, or maxFeePerGas and maxPriorityFeePerGas.

Please double check your transactions' parameters.`,
            shouldBeReported: false,
        },
        PERSONALSIGN_MISSING_ADDRESS_PARAM: {
            number: 116,
            message: 'Missing "address" param when calling personal_sign.',
            title: "Missing `address` param when calling personal_sign.",
            description: `You called \`personal_sign\` with incorrect parameters.

Please check that you are sending an \`address\` parameter.`,
            shouldBeReported: false,
        },
        EMPTY_URL: {
            number: 117,
            message: "Empty string `%value%` for network or forking URL - Expected a non-empty string.",
            title: "Empty string `%value%` for network or forking URL - Expected a non-empty string.",
            description: `You are trying to connect to a network with an empty network or forking URL.

Please check that you are sending a non-empty string for network or forking \`URL\` parameter.`,
            shouldBeReported: false,
        },
        INCOMPATIBLE_EIP7702_FIELDS: {
            number: 118,
            message: "An incompatible transaction with gasPrice and EIP-7702 fields.",
            title: "Incompatible EIP-7702 parameters",
            description: `You are trying to send a transaction with a locally managed
account, and its parameters are incompatible. You sent both gasPrice and authorizationList.

Please double check your transactions' parameters.`,
            shouldBeReported: false,
        },
        DATA_FIELD_CANNOT_BE_NULL_WITH_NULL_ADDRESS: {
            number: 119,
            message: `The "to" field is undefined, and the "data" field is also undefined; however, a transaction to the null address cannot have an undefined "data" field.`,
            title: "Transaction to null address cannot have undefined data",
            description: "The transaction to the null address cannot have undefined data",
            shouldBeReported: false,
        },
    },
    TASK_DEFINITIONS: {
        PARAM_AFTER_VARIADIC: {
            number: 200,
            message: "Could not set positional param %paramName% for task %taskName% because there is already a variadic positional param and it has to be the last positional one.",
            title: "Could not add positional param",
            description: `Could add a positional param to your task because
there is already a variadic positional param and it has to be the last
positional one.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        PARAM_ALREADY_DEFINED: {
            number: 201,
            message: "Could not set param %paramName% for task %taskName% because its name is already used.",
            title: "Repeated param name",
            description: `Could not add a param to your task because its name is already used.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        PARAM_CLASHES_WITH_HARDHAT_PARAM: {
            number: 202,
            message: "Could not set param %paramName% for task %taskName% because its name is used as a param for Hardhat.",
            title: "Hardhat and task param names clash",
            description: `Could not add a param to your task because its name is used as a param for Hardhat.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        MANDATORY_PARAM_AFTER_OPTIONAL: {
            number: 203,
            message: "Could not set param %paramName% for task %taskName% because it is mandatory and it was added after an optional positional param.",
            title: "Optional param followed by a required one",
            description: `Could not add param to your task because it is required and it was added after an optional positional param.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        ACTION_NOT_SET: {
            number: 204,
            message: "No action set for task %taskName%.",
            title: "Tried to run task without an action",
            description: `A task was run, but it has no action set.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        RUNSUPER_NOT_AVAILABLE: {
            number: 205,
            message: "Tried to call runSuper from a non-overridden definition of task %taskName%",
            title: "`runSuper` not available",
            description: `You tried to call \`runSuper\` from a non-overridden task.

Please use \`runSuper.isDefined\` to make sure that you can call it.`,
            shouldBeReported: false,
        },
        DEFAULT_VALUE_WRONG_TYPE: {
            number: 206,
            message: "Default value for param %paramName% of task %taskName% doesn't match its expected type.",
            title: "Default value has incorrect type",
            description: `One of your tasks has a parameter whose default value doesn't match the expected type.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        DEFAULT_IN_MANDATORY_PARAM: {
            number: 207,
            message: "Default value for param %paramName% of task %taskName% shouldn't be set.",
            title: "Required parameter has a default value",
            description: `One of your tasks has a required parameter with a default value.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        INVALID_PARAM_NAME_CASING: {
            number: 208,
            message: "Invalid param name %paramName% in task %taskName%. Param names must be camelCase.",
            title: "Invalid casing in parameter name",
            description: `Your parameter names must use camelCase.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        OVERRIDE_NO_MANDATORY_PARAMS: {
            number: 209,
            message: "Redefinition of task %taskName% failed. Unsupported operation adding mandatory (non optional) param definitions in an overridden task.",
            title: "Attempted to add mandatory params to an overridden task",
            description: `You can't add mandatory (non optional) param definitions in an overridden task.
The only supported param additions for overridden tasks are flags
and optional params.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        OVERRIDE_NO_POSITIONAL_PARAMS: {
            number: 210,
            message: "Redefinition of task %taskName% failed. Unsupported operation adding positional param definitions in an overridden task.",
            title: "Attempted to add positional params to an overridden task",
            description: `You can't add positional param definitions in an overridden task.
The only supported param additions for overridden tasks are flags
and optional params.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        OVERRIDE_NO_VARIADIC_PARAMS: {
            number: 211,
            message: "Redefinition of task %taskName% failed. Unsupported operation adding variadic param definitions in an overridden task.",
            title: "Attempted to add variadic params to an overridden task",
            description: `You can't add variadic param definitions in an overridden task.
The only supported param additions for overridden tasks are flags
and optional params.

Please double check your task definitions.`,
            shouldBeReported: false,
        },
        CLI_ARGUMENT_TYPE_REQUIRED: {
            number: 212,
            title: "Invalid argument type",
            message: "Task %task% is not a subtask but one of its arguments uses the type %type%, which is not parseable.",
            description: `Tasks that can be invoked from the command line require CLIArgumentType types for their arguments.

What makes these types special is that they can be represented as strings, so you can write them down in the terminal.`,
            shouldBeReported: false,
        },
        TASK_SCOPE_CLASH: {
            number: 213,
            message: "A clash was found while creating scope '%scopeName%', since a task with that name already exists.",
            title: "Attempted to create a scope with a name already used by a task",
            description: `You can't create a scope if a task with that name already exists.
Please double check your task definitions.`,
            shouldBeReported: false,
        },
        SCOPE_TASK_CLASH: {
            number: 214,
            message: "A clash was found while creating task '%taskName%', since a scope with that name already exists.",
            title: "Attempted to create a task with a name already used by a scope",
            description: `You can't create a task if a scope with that name already exists.
Please double check your task definitions.`,
            shouldBeReported: false,
        },
        DEPRECATED_TRANSFORM_IMPORT_TASK: {
            number: 215,
            title: "Use of deprecated remapping task",
            message: "Task TASK_COMPILE_TRANSFORM_IMPORT_NAME is deprecated. Please update your @nomicfoundation/hardhat-foundry plugin version.",
            description: `This task has been deprecated in favor of a new approach.`,
            shouldBeReported: true,
        },
    },
    ARGUMENTS: {
        INVALID_ENV_VAR_VALUE: {
            number: 300,
            message: "Invalid environment variable '%varName%' with value: '%value%'",
            title: "Invalid environment variable value",
            description: `You are setting one of Hardhat's arguments using an environment variable, but it has an incorrect value.

Please double check your environment variables.`,
            shouldBeReported: false,
        },
        INVALID_VALUE_FOR_TYPE: {
            number: 301,
            message: "Invalid value %value% for argument %name% of type %type%",
            title: "Invalid argument type",
            description: `One of your Hardhat or task arguments has an invalid type.

Please double check your arguments.`,
            shouldBeReported: false,
        },
        INVALID_INPUT_FILE: {
            number: 302,
            message: "Invalid argument %name%: File %value% doesn't exist or is not a readable file.",
            title: "Invalid file argument",
            description: `One of your tasks expected a file as an argument, but you provided a
nonexistent or non-readable file.

Please double check your arguments.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_TASK: {
            number: 303,
            message: "Unrecognized task '%task%'",
            title: "Unrecognized task",
            description: `Tried to run a nonexistent task.

Please double check the name of the task you are trying to run.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_COMMAND_LINE_ARG: {
            number: 304,
            message: "Unrecognised command line argument %argument%.\nNote that task arguments must come after the task name.",
            title: "Unrecognized command line argument",
            description: `Hardhat couldn't recognize one of your command line arguments.

This may be because you are writing it before the task name. It should come after it.

Please double check how you invoked Hardhat.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_PARAM_NAME: {
            number: 305,
            message: "Unrecognized param %param%",
            title: "Unrecognized param",
            description: `Hardhat couldn't recognize one of your tasks' parameters.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        MISSING_TASK_ARGUMENT: {
            number: 306,
            message: "The '%param%' parameter of task '%task%' expects a value, but none was passed.",
            title: "Missing task argument",
            description: `You tried to run a task, but one of its required arguments was missing.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        MISSING_POSITIONAL_ARG: {
            number: 307,
            message: "Missing positional argument %param%",
            title: "Missing task positional argument",
            description: `You tried to run a task, but one of its required arguments was missing.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_POSITIONAL_ARG: {
            number: 308,
            message: "Unrecognized positional argument %argument%",
            title: "Unrecognized task positional argument",
            description: `You tried to run a task with more positional arguments than expected.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        REPEATED_PARAM: {
            number: 309,
            message: "Repeated parameter %param%",
            title: "Repeated task parameter",
            description: `You tried to run a task with a repeated parameter.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        PARAM_NAME_INVALID_CASING: {
            number: 310,
            message: "Invalid param %param%. Command line params must be lowercase.",
            title: "Invalid casing in command line parameter",
            description: `You tried to run hardhat with a parameter with invalid casing. They must be lowercase.

Please double check how you invoked Hardhat.`,
            shouldBeReported: false,
        },
        INVALID_JSON_ARGUMENT: {
            number: 311,
            message: "Error parsing JSON value for argument %param%: %error%",
            title: "Invalid JSON parameter",
            description: `You tried to run a task with an invalid JSON parameter.

Please double check how you invoked Hardhat or ran your task.`,
            shouldBeReported: false,
        },
        RUNNING_SUBTASK_FROM_CLI: {
            number: 312,
            title: "Subtask run from the command line",
            message: "Trying to run the %name% subtask from the CLI",
            description: `You tried to run a subtask from the command line.

This is not supported. Please run the help task to see the available options.`,
            shouldBeReported: false,
        },
        TYPECHECK_USED_IN_JAVASCRIPT_PROJECT: {
            number: 313,
            title: "The --typecheck flag was used in a javascript project",
            message: "Trying to use the --typecheck flag, but the project is not in typescript",
            description: `You tried to run Hardhat with the \`--typecheck\` flag in a javascript project.

This flag can only be used in typescript projects.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_SCOPE: {
            number: 314,
            message: "Unrecognized scope '%scope%'",
            title: "Unrecognized scope",
            description: `Tried to run a task from a nonexistent scope.

Please double check the scope of the task you are trying to run.`,
            shouldBeReported: false,
        },
        UNRECOGNIZED_SCOPED_TASK: {
            number: 315,
            message: "Unrecognized task '%task%' under scope '%scope%'",
            title: "Unrecognized scoped task",
            description: `Tried to run a nonexistent scoped task.

Please double check the name of the task you are trying to run.`,
            shouldBeReported: false,
        },
    },
    RESOLVER: {
        FILE_NOT_FOUND: {
            number: 400,
            message: "File %file% doesn't exist.",
            title: "Solidity file not found",
            description: `Tried to resolve a nonexistent Solidity file as an entry-point.`,
            shouldBeReported: false,
        },
        LIBRARY_NOT_INSTALLED: {
            number: 401,
            message: "Library %library% is not installed.",
            title: "Solidity library not installed",
            description: `One of your Solidity sources imports a library that is not installed.

Please double check your imports or install the missing dependency.`,
            shouldBeReported: false,
        },
        LIBRARY_FILE_NOT_FOUND: {
            number: 402,
            message: "File %file% doesn't exist.",
            title: "Missing library file",
            description: `One of your libraries' files was imported but doesn't exist.

Please double check your imports or update your libraries.`,
            shouldBeReported: false,
        },
        ILLEGAL_IMPORT: {
            number: 403,
            message: "Illegal import %imported% from %from%",
            title: "Illegal Solidity import",
            description: `One of your libraries tried to use a relative import to import a file outside of its scope.

This is disabled for security reasons.`,
            shouldBeReported: false,
        },
        IMPORTED_FILE_NOT_FOUND: {
            number: 404,
            message: "File %imported%, imported from %from%, not found.",
            title: "Imported file not found",
            description: `One of your source files imported a nonexistent file.

Please double check your imports.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_BACKSLASH: {
            number: 405,
            message: "Invalid import %imported% from %from%. Imports must use / instead of \\, even in Windows",
            title: "Invalid import: use / instead of \\",
            description: `A Solidity file is trying to import another file via relative path and is using backslashes (\\\\) instead of slashes (/).

You must always use slashes (/) in Solidity imports.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_PROTOCOL: {
            number: 406,
            message: "Invalid import %imported% from %from%. Hardhat doesn't support imports via %protocol%.",
            title: "Invalid import: trying to use an unsupported protocol",
            description: `A Solidity file is trying to import a file using an unsupported protocol, like http.

You can only import files that are available locally or installed through npm.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_ABSOLUTE_PATH: {
            number: 407,
            message: "Invalid import %imported% from %from%. Hardhat doesn't support imports with absolute paths.",
            title: "Invalid import: absolute paths unsupported",
            description: `A Solidity file is trying to import a file using its absolute path.

This is not supported, as it would lead to hard-to-reproduce compilations.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_OUTSIDE_OF_PROJECT: {
            number: 408,
            message: "Invalid import %imported% from %from%. The file being imported is outside of the project",
            title: "Invalid import: file outside of the project",
            description: `A Solidity file is trying to import a file that is outside of the project.

This is not supported by Hardhat.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_WRONG_CASING: {
            number: 409,
            message: "Trying to import %imported% from %from%, but it has an incorrect casing.",
            title: "Invalid import: wrong file casing",
            description: `A Solidity file is trying to import a file but its source name casing was wrong.

Hardhat's compiler is case sensitive to ensure projects are portable across different operating systems.`,
            shouldBeReported: false,
        },
        WRONG_SOURCE_NAME_CASING: {
            number: 410,
            message: "Trying to resolve the file %incorrect% but its correct case-sensitive name is %correct%",
            title: "Incorrect source name casing",
            description: `You tried to resolve a Solidity file with an incorrect casing.

Hardhat's compiler is case sensitive to ensure projects are portable across different operating systems.`,
            shouldBeReported: false,
        },
        IMPORTED_LIBRARY_NOT_INSTALLED: {
            number: 411,
            message: "The library %library%, imported from %from%, is not installed. Try installing it using npm.",
            title: "Invalid import: library not installed",
            description: `A Solidity file is trying to import another which belongs to a library that is not installed.

Try installing the library using npm.`,
            shouldBeReported: false,
        },
        INCLUDES_OWN_PACKAGE_NAME: {
            number: 412,
            message: "Invalid import %imported% from %from%. Trying to import file using the own package's name.",
            title: "Invalid import: includes own package's name",
            description: `A Solidity file is trying to import another using its own package name. This is most likely caused by an existing symlink for the package in your node_modules.

Use a relative import instead of referencing the package's name.`,
            shouldBeReported: false,
        },
        IMPORTED_MAPPED_FILE_NOT_FOUND: {
            number: 413,
            message: "File %importName% => %imported%, imported from %from%, not found.",
            title: "Imported mapped file not found",
            description: `One of your source files imported a nonexistent or not installed file.

Please double check your imports and installed libraries.`,
            shouldBeReported: false,
        },
        INVALID_IMPORT_OF_DIRECTORY: {
            number: 414,
            message: "Invalid import %imported% from %from%. Attempting to import a directory. Directories cannot be imported.",
            title: "Invalid import: a directory cannot be imported",
            description: `A Solidity file is attempting to import a directory, which is not possible.

Please double check your imports.`,
            shouldBeReported: false,
        },
        AMBIGUOUS_SOURCE_NAMES: {
            number: 415,
            message: "Two different source names (%sourcenames%) resolve to the same file (%file%).",
            title: "Ambiguous source names",
            description: `Two different source names map to the same file.

This is probably caused by multiple remappings pointing to the same source file.`,
            shouldBeReported: false,
        },
    },
    SOLC: {
        INVALID_VERSION: {
            number: 500,
            message: `Solidity version %version% is invalid or hasn't been released yet.

If you are certain it has been released, run "npx hardhat clean --global" and try again`,
            title: "Invalid or unreleased `solc` version",
            description: `The Solidity version in your config is invalid or hasn't been released yet.

If you are certain it has been released, run \`npx hardhat clean --global\` and try again.`,
            shouldBeReported: false,
        },
        DOWNLOAD_FAILED: {
            number: 501,
            message: "Couldn't download compiler version %remoteVersion%. Please check your internet connection and try again.",
            title: "`solc` download failed",
            description: `Couldn't download \`solc\`.

Please check your internet connection and try again.`,
            shouldBeReported: false,
        },
        VERSION_LIST_DOWNLOAD_FAILED: {
            number: 502,
            message: "Couldn't download compiler version list. Please check your internet connection and try again.",
            title: "Couldn't obtain `solc` version list",
            description: `Couldn't download \`solc\`'s version list.

Please check your internet connection and try again.`,
            shouldBeReported: false,
        },
        INVALID_DOWNLOAD: {
            number: 503,
            message: `Couldn't download compiler version %remoteVersion%: Checksum verification failed.

Please check your internet connection and try again.

If this error persists, run "npx hardhat clean --global".`,
            title: "Downloaded `solc` checksum verification failed",
            description: `Hardhat downloaded a version of the Solidity compiler, and its checksum verification failed.

Please check your internet connection and try again.

If this error persists, run \`npx hardhat clean --global\`.`,
            shouldBeReported: false,
        },
        CANT_GET_COMPILER: {
            number: 504,
            message: "The solc compiler couldn't be obtained for version %version%",
            title: "The solc compiler couldn't be obtained",
            description: `Hardhat couldn't obtain a valid solc compiler.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        CANT_RUN_NATIVE_COMPILER: {
            number: 505,
            message: `A native version of solc failed to run.

If you are running MacOS, try installing Apple Rosetta.

If this error persists, run "npx hardhat clean --global".`,
            title: "Failed to run native solc",
            description: `Hardhat successfully downloaded a native version of solc but it doesn't run.

If you are running MacOS, try installing Apple Rosetta.

If this error persists, run "npx hardhat clean --global".`,
            shouldBeReported: false,
        },
        SOLCJS_ERROR: {
            number: 506,
            message: "Error running solcjs: %error%",
            title: "Error running solcjs",
            description: `There was an error while running the solcjs compiler.`,
            shouldBeReported: false,
        },
    },
    BUILTIN_TASKS: {
        COMPILE_FAILURE: {
            number: 600,
            message: "Compilation failed",
            title: "Compilation failed",
            description: `Your smart contracts failed to compile.

Please check Hardhat's output for more details.`,
            shouldBeReported: false,
        },
        RUN_FILE_NOT_FOUND: {
            number: 601,
            message: "Script %script% doesn't exist.",
            title: "Script doesn't exist",
            description: `Tried to use \`hardhat run\` to execute a nonexistent script.

Please double check your script's path.`,
            shouldBeReported: false,
        },
        RUN_SCRIPT_ERROR: {
            number: 602,
            message: "Error running script {%script%}: %error%",
            title: "Error running script",
            description: `Running a script resulted in an error.

Please check Hardhat's output for more details.`,
            shouldBeReported: false,
        },
        FLATTEN_CYCLE: {
            number: 603,
            message: "Hardhat flatten doesn't support cyclic dependencies.",
            title: "Flatten detected cyclic dependencies",
            description: `Hardhat flatten doesn't support cyclic dependencies.

We recommend not using this kind of dependency.`,
            shouldBeReported: false,
        },
        JSONRPC_SERVER_ERROR: {
            number: 604,
            message: "Error running JSON-RPC server: %error%",
            title: "Error running JSON-RPC server",
            description: `There was an error while starting the JSON-RPC HTTP server.`,
            shouldBeReported: false,
        },
        JSONRPC_UNSUPPORTED_NETWORK: {
            number: 605,
            message: "Unsupported network for JSON-RPC server. Only hardhat is currently supported.",
            title: "Unsupported network for JSON-RPC server.",
            description: `JSON-RPC server can only be started when running the Hardhat Network.

To start the JSON-RPC server, retry the command without the --network parameter.`,
            shouldBeReported: false,
        },
        COMPILATION_JOBS_CREATION_FAILURE: {
            number: 606,
            message: `The project cannot be compiled, see reasons below.

%reasons%`,
            title: "The project cannot be compiled",
            description: `The project cannot be compiled with the current settings.`,
            shouldBeReported: false,
        },
        NODE_FORK_BLOCK_NUMBER_WITHOUT_URL: {
            number: 607,
            message: `You specified a fork block number but not an URL.`,
            title: "Missing fork URL",
            description: `You passed a block number to fork from, but not a URL. Hardhat cannot fork
if the URL of the JSON-RPC wasn't set.`,
            shouldBeReported: false,
        },
        COMPILE_TASK_UNSUPPORTED_SOLC_VERSION: {
            number: 608,
            message: `Version %version% is not supported by Hardhat.

The first supported version is %firstSupportedVersion%`,
            title: "Unsupported solc version",
            description: `This version of solidity is not supported by Hardhat.
Please use a newer, supported version.`,
            shouldBeReported: true,
        },
        TEST_TASK_ESM_TESTS_RUN_TWICE: {
            number: 609,
            message: `Your project uses ESM and you've programmatically run your tests twice. This is not supported yet.`,
            title: "Running tests twice in an ESM project",
            description: 'You have run your tests twice programmatically and your project is an ESM project (you have `"type": "module"` in your `package.json`, or some of your files have the `.mjs` extension). This is not supported by Mocha yet.',
            shouldBeReported: true,
        },
    },
    ARTIFACTS: {
        NOT_FOUND: {
            number: 700,
            message: 'Artifact for contract "%contractName%" not found. %suggestion%',
            title: "Artifact not found",
            description: `Tried to import a nonexistent artifact.

Please double check that your contracts have been compiled and double check your artifact's name.`,
            shouldBeReported: false,
        },
        MULTIPLE_FOUND: {
            number: 701,
            message: `There are multiple artifacts for contract "%contractName%", please use a fully qualified name.

Please replace %contractName% for one of these options wherever you are trying to read its artifact:

%candidates%
`,
            title: "Multiple artifacts found",
            description: `There are multiple artifacts that match the given contract name, and Hardhat doesn't know which one to use.

Please use the fully qualified name of the contract to disambiguate it.`,
            shouldBeReported: false,
        },
        WRONG_CASING: {
            number: 702,
            message: "Invalid artifact path %incorrect%, its correct case-sensitive path is %correct%",
            title: "Incorrect artifact path casing",
            description: `You tried to get an artifact file with an incorrect casing.

Hardhat's artifact resolution is case sensitive to ensure projects are portable across different operating systems.`,
            shouldBeReported: true,
        },
    },
    PLUGINS: {
        BUIDLER_PLUGIN: {
            number: 800,
            message: `You are using %plugin%, which is a Buidler plugin. Use the equivalent
Hardhat plugin instead.`,
            title: "Using a buidler plugin",
            description: `You are trying to use a Buidler plugin in Hardhat. This is not supported.

Please use the equivalent Hardhat plugin instead.`,
            shouldBeReported: false,
        },
        MISSING_DEPENDENCIES: {
            number: 801,
            message: `Plugin %plugin% requires the following dependencies to be installed: %missingDependencies%.
Please run: npm install --save-dev %missingDependenciesVersions%`,
            title: "Plugin dependencies not installed",
            description: `You are trying to use a plugin with unmet dependencies.

Please follow Hardhat's instructions to resolve this.`,
            shouldBeReported: false,
        },
    },
    INTERNAL: {
        TEMPLATE_INVALID_VARIABLE_NAME: {
            number: 900,
            message: "Variable names can only include ascii letters and numbers, and start with a letter, but got %variable%",
            title: "Invalid error message template",
            description: `An error message template contains an invalid variable name. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        TEMPLATE_VALUE_CONTAINS_VARIABLE_TAG: {
            number: 901,
            message: "Template values can't include variable tags, but %variable%'s value includes one",
            title: "Invalid error message replacement",
            description: `Tried to replace an error message variable with a value that contains another variable name. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        TEMPLATE_VARIABLE_TAG_MISSING: {
            number: 902,
            message: "Variable %variable%'s tag not present in the template",
            title: "Missing replacement value from error message template",
            description: `An error message template is missing a replacement value. This is a bug.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
        WRONG_ARTIFACT_PATH: {
            number: 903,
            message: "The inferred artifact path for contract %contractName% is %artifactPath%, but this file doesn't exist",
            title: "Inferred artifact path doesn't exist",
            description: `The inferred artifact path doesn't exist.

Please [report it](https://github.com/nomiclabs/hardhat/issues/new) to help us improve Hardhat.`,
            shouldBeReported: true,
        },
    },
    SOURCE_NAMES: {
        INVALID_SOURCE_NAME_ABSOLUTE_PATH: {
            number: 1000,
            message: "Invalid source name %name%. Expected source name but found an absolute path.",
            title: "Invalid source name: absolute path",
            description: `A Solidity source name was expected, but an absolute path was given.

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
        INVALID_SOURCE_NAME_RELATIVE_PATH: {
            number: 1001,
            message: "Invalid source name %name%. Expected source name but found a relative path.",
            title: "Invalid source name: relative path",
            description: `A Solidity source name was expected, but a relative path was given.

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
        INVALID_SOURCE_NAME_BACKSLASHES: {
            number: 1002,
            message: "Invalid source %name%. The source name uses backslashes (\\) instead of slashes (/).",
            title: "Invalid source name: backslashes",
            description: `A Solidity source name was invalid because it uses backslashes (\\\\) instead of slashes (/).

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
        INVALID_SOURCE_NOT_NORMALIZED: {
            number: 1003,
            message: "Invalid source name %name%. Source names must be normalized",
            title: "Invalid source name: not normalized",
            description: `A Solidity source name was invalid because it wasn't normalized. It probably contains some "." or "..".

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
        WRONG_CASING: {
            number: 1004,
            message: "Invalid source map %incorrect%, its correct case-sensitive source name is %correct%",
            title: "Incorrect source name casing",
            description: `You tried to resolve a Solidity file with an incorrect casing.

Hardhat's compiler is case sensitive to ensure projects are portable across different operating systems.`,
            shouldBeReported: true,
        },
        FILE_NOT_FOUND: {
            number: 1005,
            message: "Solidity source file %name% not found",
            title: "Solidity source file not found",
            description: `A source name should correspond to an existing Solidity file but it doesn't.

Hardhat's compiler is case sensitive to ensure projects are portable across different operating systems.`,
            shouldBeReported: true,
        },
        NODE_MODULES_AS_LOCAL: {
            number: 1006,
            message: "The file %path% is treated as local but is inside a node_modules directory",
            title: "File from node_modules treated as local",
            description: `A file was treated as local but is inside a node_modules directory.

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
        EXTERNAL_AS_LOCAL: {
            number: 1007,
            message: "The file %path% is treated as local but is outside the project",
            title: "File from outside the project treated as local",
            description: `A file was treated as local but is outside the project.

If you aren't overriding compilation-related tasks, please report this as a bug.`,
            shouldBeReported: true,
        },
    },
    CONTRACT_NAMES: {
        INVALID_FULLY_QUALIFIED_NAME: {
            number: 1100,
            message: "Invalid fully qualified contract name %name%.",
            title: "Invalid fully qualified contract name",
            description: `A contract name was expected to be in fully qualified form, but it's not.

A fully qualified name should look like file.sol:Contract`,
            shouldBeReported: false,
        },
    },
    VARS: {
        ONLY_MANAGED_IN_CLI: {
            number: 1200,
            title: "Configuration variables can only be managed from the CLI",
            message: "Configuration variables can only be managed from the CLI. They cannot be modified programmatically.",
            description: `Configuration variables can only be managed from the CLI. They cannot be modified programmatically.`,
            shouldBeReported: false,
        },
        VALUE_NOT_FOUND_FOR_VAR: {
            number: 1201,
            title: "Configuration variable is not set",
            message: "Cannot find a value for the configuration variable '%value%'. Use 'npx hardhat vars set %value%' to set it or 'npx hardhat vars setup' to list all the configuration variables used by this project.",
            description: `Cannot find a value for a mandatory configuration variable.

Use 'npx hardhat vars set VAR' to set it or 'npx hardhat vars setup' to list all the configuration variables used by this project.`,
            shouldBeReported: false,
        },
        INVALID_CONFIG_VAR_NAME: {
            number: 1202,
            title: "Invalid name for a configuration variable",
            message: "Invalid name for a configuration variable: '%value%'. Configuration variables can only have alphanumeric characters and underscores, and they cannot start with a number.",
            description: `Invalid name for a configuration variable.

Configuration variables can only have alphanumeric characters and underscores, and they cannot start with a number.`,
            shouldBeReported: false,
        },
        INVALID_EMPTY_VALUE: {
            number: 1203,
            title: "Invalid empty value for configuration variable",
            message: "A configuration variable cannot have an empty value.",
            description: "A configuration variable cannot have an empty value.",
            shouldBeReported: false,
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