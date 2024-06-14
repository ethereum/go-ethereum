"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const chalk_1 = __importDefault(require("chalk"));
const debug_1 = __importDefault(require("debug"));
const fs_extra_1 = __importDefault(require("fs-extra"));
const constants_1 = require("../internal/constants");
const config_env_1 = require("../internal/core/config/config-env");
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
const construction_1 = require("../internal/core/providers/construction");
const util_1 = require("../internal/core/providers/util");
const server_1 = require("../internal/hardhat-network/jsonrpc/server");
const reporter_1 = require("../internal/sentry/reporter");
const default_config_1 = require("../internal/core/config/default-config");
const task_names_1 = require("./task-names");
const watch_1 = require("./utils/watch");
const log = (0, debug_1.default)("hardhat:core:tasks:node");
function printDefaultConfigWarning() {
    console.log(chalk_1.default.bold("WARNING: These accounts, and their private keys, are publicly known."));
    console.log(chalk_1.default.bold("Any funds sent to them on Mainnet or any other live network WILL BE LOST."));
}
function logHardhatNetworkAccounts(networkConfig) {
    const isDefaultConfig = !Array.isArray(networkConfig.accounts) &&
        networkConfig.accounts.mnemonic === default_config_1.HARDHAT_NETWORK_MNEMONIC;
    const { bytesToHex: bufferToHex, privateToAddress, toBytes, toChecksumAddress, } = require("@nomicfoundation/ethereumjs-util");
    console.log("Accounts");
    console.log("========");
    if (isDefaultConfig) {
        console.log();
        printDefaultConfigWarning();
        console.log();
    }
    const accounts = (0, util_1.normalizeHardhatNetworkAccountsConfig)(networkConfig.accounts);
    for (const [index, account] of accounts.entries()) {
        const address = toChecksumAddress(bufferToHex(privateToAddress(toBytes(account.privateKey))));
        const balance = (BigInt(account.balance) / 10n ** 18n).toString(10);
        let entry = `Account #${index}: ${address} (${balance} ETH)`;
        if (isDefaultConfig) {
            const privateKey = bufferToHex(toBytes(account.privateKey));
            entry += `
Private Key: ${privateKey}`;
        }
        console.log(entry);
        console.log();
    }
    if (isDefaultConfig) {
        printDefaultConfigWarning();
        console.log();
    }
}
(0, config_env_1.subtask)(task_names_1.TASK_NODE_GET_PROVIDER)
    .addOptionalParam("forkUrl", undefined, undefined, config_env_1.types.string)
    .addOptionalParam("forkBlockNumber", undefined, undefined, config_env_1.types.int)
    .setAction(async ({ forkBlockNumber: forkBlockNumberParam, forkUrl: forkUrlParam, }, { artifacts, config, network, userConfig }) => {
    let provider = network.provider;
    if (network.name !== constants_1.HARDHAT_NETWORK_NAME) {
        log(`Creating hardhat provider for JSON-RPC server`);
        provider = await (0, construction_1.createProvider)(config, constants_1.HARDHAT_NETWORK_NAME, artifacts);
    }
    const hardhatNetworkConfig = config.networks[constants_1.HARDHAT_NETWORK_NAME];
    const forkUrlConfig = hardhatNetworkConfig.forking?.url;
    const forkBlockNumberConfig = hardhatNetworkConfig.forking?.blockNumber;
    const forkUrl = forkUrlParam ?? forkUrlConfig;
    const forkBlockNumber = forkBlockNumberParam ?? forkBlockNumberConfig;
    // we throw an error if the user specified a forkBlockNumber but not a
    // forkUrl
    if (forkBlockNumber !== undefined && forkUrl === undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.NODE_FORK_BLOCK_NUMBER_WITHOUT_URL);
    }
    // if the url or the block is different to the one in the configuration,
    // we use hardhat_reset to set the fork
    if (forkUrl !== forkUrlConfig ||
        forkBlockNumber !== forkBlockNumberConfig) {
        await provider.request({
            method: "hardhat_reset",
            params: [
                {
                    forking: {
                        jsonRpcUrl: forkUrl,
                        blockNumber: forkBlockNumber,
                    },
                },
            ],
        });
    }
    const hardhatNetworkUserConfig = userConfig.networks?.[constants_1.HARDHAT_NETWORK_NAME] ?? {};
    // enable logging
    await provider.request({
        method: "hardhat_setLoggingEnabled",
        params: [hardhatNetworkUserConfig.loggingEnabled ?? true],
    });
    return provider;
});
(0, config_env_1.subtask)(task_names_1.TASK_NODE_CREATE_SERVER)
    .addParam("hostname", undefined, undefined, config_env_1.types.string)
    .addParam("port", undefined, undefined, config_env_1.types.int)
    .addParam("provider", undefined, undefined, config_env_1.types.any)
    .setAction(async ({ hostname, port, provider, }) => {
    const serverConfig = {
        hostname,
        port,
        provider,
    };
    const server = new server_1.JsonRpcServer(serverConfig);
    return server;
});
/**
 * This task will be called when the server was successfully created, but it's
 * not ready for receiving requests yet.
 */
(0, config_env_1.subtask)(task_names_1.TASK_NODE_SERVER_CREATED)
    .addParam("hostname", undefined, undefined, config_env_1.types.string)
    .addParam("port", undefined, undefined, config_env_1.types.int)
    .addParam("provider", undefined, undefined, config_env_1.types.any)
    .addParam("server", undefined, undefined, config_env_1.types.any)
    .setAction(async ({}) => {
    // this task is meant to be overriden by plugin writers
});
/**
 * This subtask will be run when the server is ready to accept requests
 */
(0, config_env_1.subtask)(task_names_1.TASK_NODE_SERVER_READY)
    .addParam("address", undefined, undefined, config_env_1.types.string)
    .addParam("port", undefined, undefined, config_env_1.types.int)
    .addParam("provider", undefined, undefined, config_env_1.types.any)
    .addParam("server", undefined, undefined, config_env_1.types.any)
    .setAction(async ({ address, port, }, { config }) => {
    console.log(chalk_1.default.green(`Started HTTP and WebSocket JSON-RPC server at http://${address}:${port}/`));
    console.log();
    const networkConfig = config.networks[constants_1.HARDHAT_NETWORK_NAME];
    logHardhatNetworkAccounts(networkConfig);
});
(0, config_env_1.task)(task_names_1.TASK_NODE, "Starts a JSON-RPC server on top of Hardhat Network")
    .addOptionalParam("hostname", "The host to which to bind to for new connections (Defaults to 127.0.0.1 running locally, and 0.0.0.0 in Docker)", undefined, config_env_1.types.string)
    .addOptionalParam("port", "The port on which to listen for new connections", 8545, config_env_1.types.int)
    .addOptionalParam("fork", "The URL of the JSON-RPC server to fork from", undefined, config_env_1.types.string)
    .addOptionalParam("forkBlockNumber", "The block number to fork from", undefined, config_env_1.types.int)
    .setAction(async ({ forkBlockNumber, fork: forkUrl, hostname: hostnameParam, port, }, { config, hardhatArguments, network, run }) => {
    // we throw if the user specified a network argument and it's not hardhat
    if (network.name !== constants_1.HARDHAT_NETWORK_NAME &&
        hardhatArguments.network !== undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.JSONRPC_UNSUPPORTED_NETWORK);
    }
    try {
        const provider = await run(task_names_1.TASK_NODE_GET_PROVIDER, {
            forkBlockNumber,
            forkUrl,
        });
        // the default hostname is "127.0.0.1" unless we are inside a docker
        // container, in that case we use "0.0.0.0"
        let hostname;
        if (hostnameParam !== undefined) {
            hostname = hostnameParam;
        }
        else {
            const insideDocker = fs_extra_1.default.existsSync("/.dockerenv");
            if (insideDocker) {
                hostname = "0.0.0.0";
            }
            else {
                hostname = "127.0.0.1";
            }
        }
        const server = await run(task_names_1.TASK_NODE_CREATE_SERVER, {
            hostname,
            port,
            provider,
        });
        await run(task_names_1.TASK_NODE_SERVER_CREATED, {
            hostname,
            port,
            provider,
            server,
        });
        const { port: actualPort, address } = await server.listen();
        let watcher;
        try {
            watcher = await (0, watch_1.watchCompilerOutput)(provider, config.paths);
        }
        catch (error) {
            console.warn(chalk_1.default.yellow("There was a problem watching the compiler output, changes in the contracts won't be reflected in the Hardhat Network. Run Hardhat with --verbose to learn more."));
            log("Compilation output can't be watched. Please report this to help us improve Hardhat.\n", error);
            if (error instanceof Error) {
                reporter_1.Reporter.reportError(error);
            }
        }
        await run(task_names_1.TASK_NODE_SERVER_READY, {
            address,
            port: actualPort,
            provider,
            server,
        });
        await server.waitUntilClosed();
        await watcher?.close();
    }
    catch (error) {
        if (errors_1.HardhatError.isHardhatError(error)) {
            throw error;
        }
        if (error instanceof Error) {
            throw new errors_1.HardhatError(errors_list_1.ERRORS.BUILTIN_TASKS.JSONRPC_SERVER_ERROR, {
                error: error.message,
            }, error);
        }
        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
    }
});
//# sourceMappingURL=node.js.map