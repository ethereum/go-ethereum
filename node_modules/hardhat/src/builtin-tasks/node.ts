import type EthereumjsUtilT from "@nomicfoundation/ethereumjs-util";

import chalk from "chalk";
import debug from "debug";
import fsExtra from "fs-extra";

import { HARDHAT_NETWORK_NAME } from "../internal/constants";
import { subtask, task, types } from "../internal/core/config/config-env";
import { HardhatError } from "../internal/core/errors";
import { ERRORS } from "../internal/core/errors-list";
import { createProvider } from "../internal/core/providers/construction";
import { normalizeHardhatNetworkAccountsConfig } from "../internal/core/providers/util";
import {
  JsonRpcServer as JsonRpcServerImpl,
  JsonRpcServerConfig,
} from "../internal/hardhat-network/jsonrpc/server";
import { Reporter } from "../internal/sentry/reporter";
import {
  EthereumProvider,
  HardhatNetworkConfig,
  JsonRpcServer,
} from "../types";

import { HARDHAT_NETWORK_MNEMONIC } from "../internal/core/config/default-config";
import {
  TASK_NODE,
  TASK_NODE_CREATE_SERVER,
  TASK_NODE_GET_PROVIDER,
  TASK_NODE_SERVER_CREATED,
  TASK_NODE_SERVER_READY,
} from "./task-names";
import { watchCompilerOutput, Watcher } from "./utils/watch";

const log = debug("hardhat:core:tasks:node");

function printDefaultConfigWarning() {
  console.log(
    chalk.bold(
      "WARNING: These accounts, and their private keys, are publicly known."
    )
  );
  console.log(
    chalk.bold(
      "Any funds sent to them on Mainnet or any other live network WILL BE LOST."
    )
  );
}

function logHardhatNetworkAccounts(networkConfig: HardhatNetworkConfig) {
  const isDefaultConfig =
    !Array.isArray(networkConfig.accounts) &&
    networkConfig.accounts.mnemonic === HARDHAT_NETWORK_MNEMONIC;

  const {
    bytesToHex: bufferToHex,
    privateToAddress,
    toBytes,
    toChecksumAddress,
  } = require("@nomicfoundation/ethereumjs-util") as typeof EthereumjsUtilT;

  console.log("Accounts");
  console.log("========");

  if (isDefaultConfig) {
    console.log();
    printDefaultConfigWarning();
    console.log();
  }

  const accounts = normalizeHardhatNetworkAccountsConfig(
    networkConfig.accounts
  );

  for (const [index, account] of accounts.entries()) {
    const address = toChecksumAddress(
      bufferToHex(privateToAddress(toBytes(account.privateKey)))
    );

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

subtask(TASK_NODE_GET_PROVIDER)
  .addOptionalParam("forkUrl", undefined, undefined, types.string)
  .addOptionalParam("forkBlockNumber", undefined, undefined, types.int)
  .setAction(
    async (
      {
        forkBlockNumber: forkBlockNumberParam,
        forkUrl: forkUrlParam,
      }: {
        forkBlockNumber?: number;
        forkUrl?: string;
      },
      { artifacts, config, network, userConfig }
    ): Promise<EthereumProvider> => {
      let provider = network.provider;

      if (network.name !== HARDHAT_NETWORK_NAME) {
        log(`Creating hardhat provider for JSON-RPC server`);
        provider = await createProvider(
          config,
          HARDHAT_NETWORK_NAME,
          artifacts
        );
      }

      const hardhatNetworkConfig = config.networks[HARDHAT_NETWORK_NAME];

      const forkUrlConfig = hardhatNetworkConfig.forking?.url;
      const forkBlockNumberConfig = hardhatNetworkConfig.forking?.blockNumber;

      const forkUrl = forkUrlParam ?? forkUrlConfig;
      const forkBlockNumber = forkBlockNumberParam ?? forkBlockNumberConfig;

      // we throw an error if the user specified a forkBlockNumber but not a
      // forkUrl
      if (forkBlockNumber !== undefined && forkUrl === undefined) {
        throw new HardhatError(
          ERRORS.BUILTIN_TASKS.NODE_FORK_BLOCK_NUMBER_WITHOUT_URL
        );
      }

      // if the url or the block is different to the one in the configuration,
      // we use hardhat_reset to set the fork
      if (
        forkUrl !== forkUrlConfig ||
        forkBlockNumber !== forkBlockNumberConfig
      ) {
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

      const hardhatNetworkUserConfig =
        userConfig.networks?.[HARDHAT_NETWORK_NAME] ?? {};

      // enable logging
      await provider.request({
        method: "hardhat_setLoggingEnabled",
        params: [hardhatNetworkUserConfig.loggingEnabled ?? true],
      });

      return provider;
    }
  );

subtask(TASK_NODE_CREATE_SERVER)
  .addParam("hostname", undefined, undefined, types.string)
  .addParam("port", undefined, undefined, types.int)
  .addParam("provider", undefined, undefined, types.any)
  .setAction(
    async ({
      hostname,
      port,
      provider,
    }: {
      hostname: string;
      port: number;
      provider: EthereumProvider;
    }): Promise<JsonRpcServer> => {
      const serverConfig: JsonRpcServerConfig = {
        hostname,
        port,
        provider,
      };

      const server = new JsonRpcServerImpl(serverConfig);

      return server;
    }
  );

/**
 * This task will be called when the server was successfully created, but it's
 * not ready for receiving requests yet.
 */
subtask(TASK_NODE_SERVER_CREATED)
  .addParam("hostname", undefined, undefined, types.string)
  .addParam("port", undefined, undefined, types.int)
  .addParam("provider", undefined, undefined, types.any)
  .addParam("server", undefined, undefined, types.any)
  .setAction(
    async ({}: {
      hostname: string;
      port: number;
      provider: EthereumProvider;
      server: JsonRpcServer;
    }) => {
      // this task is meant to be overriden by plugin writers
    }
  );

/**
 * This subtask will be run when the server is ready to accept requests
 */
subtask(TASK_NODE_SERVER_READY)
  .addParam("address", undefined, undefined, types.string)
  .addParam("port", undefined, undefined, types.int)
  .addParam("provider", undefined, undefined, types.any)
  .addParam("server", undefined, undefined, types.any)
  .setAction(
    async (
      {
        address,
        port,
      }: {
        address: string;
        port: number;
        provider: EthereumProvider;
        server: JsonRpcServer;
      },
      { config }
    ) => {
      console.log(
        chalk.green(
          `Started HTTP and WebSocket JSON-RPC server at http://${address}:${port}/`
        )
      );

      console.log();

      const networkConfig = config.networks[HARDHAT_NETWORK_NAME];
      logHardhatNetworkAccounts(networkConfig);
    }
  );

task(TASK_NODE, "Starts a JSON-RPC server on top of Hardhat Network")
  .addOptionalParam(
    "hostname",
    "The host to which to bind to for new connections (Defaults to 127.0.0.1 running locally, and 0.0.0.0 in Docker)",
    undefined,
    types.string
  )
  .addOptionalParam(
    "port",
    "The port on which to listen for new connections",
    8545,
    types.int
  )
  .addOptionalParam(
    "fork",
    "The URL of the JSON-RPC server to fork from",
    undefined,
    types.string
  )
  .addOptionalParam(
    "forkBlockNumber",
    "The block number to fork from",
    undefined,
    types.int
  )
  .setAction(
    async (
      {
        forkBlockNumber,
        fork: forkUrl,
        hostname: hostnameParam,
        port,
      }: {
        forkBlockNumber?: number;
        fork?: string;
        hostname?: string;
        port: number;
      },
      { config, hardhatArguments, network, run }
    ) => {
      // we throw if the user specified a network argument and it's not hardhat
      if (
        network.name !== HARDHAT_NETWORK_NAME &&
        hardhatArguments.network !== undefined
      ) {
        throw new HardhatError(
          ERRORS.BUILTIN_TASKS.JSONRPC_UNSUPPORTED_NETWORK
        );
      }

      try {
        const provider: EthereumProvider = await run(TASK_NODE_GET_PROVIDER, {
          forkBlockNumber,
          forkUrl,
        });

        // the default hostname is "127.0.0.1" unless we are inside a docker
        // container, in that case we use "0.0.0.0"
        let hostname: string;
        if (hostnameParam !== undefined) {
          hostname = hostnameParam;
        } else {
          const insideDocker = fsExtra.existsSync("/.dockerenv");
          if (insideDocker) {
            hostname = "0.0.0.0";
          } else {
            hostname = "127.0.0.1";
          }
        }

        const server: JsonRpcServer = await run(TASK_NODE_CREATE_SERVER, {
          hostname,
          port,
          provider,
        });

        await run(TASK_NODE_SERVER_CREATED, {
          hostname,
          port,
          provider,
          server,
        });

        const { port: actualPort, address } = await server.listen();

        let watcher: Watcher | undefined;
        try {
          watcher = await watchCompilerOutput(provider, config.paths);
        } catch (error) {
          console.warn(
            chalk.yellow(
              "There was a problem watching the compiler output, changes in the contracts won't be reflected in the Hardhat Network. Run Hardhat with --verbose to learn more."
            )
          );

          log(
            "Compilation output can't be watched. Please report this to help us improve Hardhat.\n",
            error
          );

          if (error instanceof Error) {
            Reporter.reportError(error);
          }
        }

        await run(TASK_NODE_SERVER_READY, {
          address,
          port: actualPort,
          provider,
          server,
        });

        await server.waitUntilClosed();
        await watcher?.close();
      } catch (error) {
        if (HardhatError.isHardhatError(error)) {
          throw error;
        }

        if (error instanceof Error) {
          throw new HardhatError(
            ERRORS.BUILTIN_TASKS.JSONRPC_SERVER_ERROR,
            {
              error: error.message,
            },
            error
          );
        }

        // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
        throw error;
      }
    }
  );
