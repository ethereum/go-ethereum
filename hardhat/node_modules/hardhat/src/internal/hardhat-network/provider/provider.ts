import type {
  Artifacts,
  EIP1193Provider,
  EthSubscription,
  HardhatNetworkChainsConfig,
  RequestArguments,
} from "../../../types";

import type {
  EdrContext,
  Provider as EdrProviderT,
  Response,
  SubscriptionEvent,
  HttpHeader,
  TracingConfigWithBuffers,
} from "@nomicfoundation/edr";
import picocolors from "picocolors";
import debug from "debug";
import { EventEmitter } from "events";
import fsExtra from "fs-extra";

import { requireNapiRsModule } from "../../../common/napi-rs";
import {
  HARDHAT_NETWORK_RESET_EVENT,
  HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT,
} from "../../constants";
import {
  InvalidArgumentsError,
  InvalidInputError,
  ProviderError,
} from "../../core/providers/errors";
import { isErrorResponse } from "../../core/providers/http";
import { getHardforkName } from "../../util/hardforks";
import { ConsoleLogger } from "../stack-traces/consoleLogger";
import { encodeSolidityStackTrace } from "../stack-traces/solidity-errors";
import { SolidityStackTrace } from "../stack-traces/solidity-stack-trace";

import { getPackageJson } from "../../util/packageInfo";
import {
  ForkConfig,
  GenesisAccount,
  IntervalMiningConfig,
  MempoolOrder,
} from "./node-types";
import {
  edrRpcDebugTraceToHardhat,
  edrTracingMessageResultToMinimalEVMResult,
  edrTracingMessageToMinimalMessage,
  edrTracingStepToMinimalInterpreterStep,
  ethereumjsIntervalMiningConfigToEdr,
  ethereumjsMempoolOrderToEdrMineOrdering,
  ethereumsjsHardforkToEdrSpecId,
} from "./utils/convertToEdr";
import { LoggerConfig, printLine, replaceLastLine } from "./modules/logger";
import { MinimalEthereumJsVm, getMinimalEthereumJsVm } from "./vm/minimal-vm";

const log = debug("hardhat:core:hardhat-network:provider");

/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */

export const DEFAULT_COINBASE = "0xc014ba5ec014ba5ec014ba5ec014ba5ec014ba5e";
let _globalEdrContext: EdrContext | undefined;

// Lazy initialize the global EDR context.
export function getGlobalEdrContext(): EdrContext {
  const { EdrContext } = requireNapiRsModule(
    "@nomicfoundation/edr"
  ) as typeof import("@nomicfoundation/edr");

  if (_globalEdrContext === undefined) {
    // Only one is allowed to exist
    _globalEdrContext = new EdrContext();
  }

  return _globalEdrContext;
}

interface HardhatNetworkProviderConfig {
  hardfork: string;
  chainId: number;
  networkId: number;
  blockGasLimit: number;
  minGasPrice: bigint;
  automine: boolean;
  intervalMining: IntervalMiningConfig;
  mempoolOrder: MempoolOrder;
  chains: HardhatNetworkChainsConfig;
  genesisAccounts: GenesisAccount[];
  allowUnlimitedContractSize: boolean;
  throwOnTransactionFailures: boolean;
  throwOnCallFailures: boolean;
  allowBlocksWithSameTimestamp: boolean;

  initialBaseFeePerGas?: number;
  initialDate?: Date;
  coinbase?: string;
  forkConfig?: ForkConfig;
  forkCachePath?: string;
  enableTransientStorage: boolean;
  enableRip7212: boolean;
}

class EdrProviderEventAdapter extends EventEmitter {}

type CallOverrideCallback = (
  address: Buffer,
  data: Buffer
) => Promise<
  { result: Buffer; shouldRevert: boolean; gas: bigint } | undefined
>;

export class EdrProviderWrapper
  extends EventEmitter
  implements EIP1193Provider
{
  private _failedStackTraces = 0;

  // temporarily added to make smock work with HH+EDR
  private _callOverrideCallback?: CallOverrideCallback;

  private constructor(
    private readonly _provider: EdrProviderT,
    // we add this for backwards-compatibility with plugins like solidity-coverage
    private readonly _node: {
      _vm: MinimalEthereumJsVm;
    }
  ) {
    super();
  }

  public static async create(
    config: HardhatNetworkProviderConfig,
    loggerConfig: LoggerConfig,
    tracingConfig?: TracingConfigWithBuffers
  ): Promise<EdrProviderWrapper> {
    const { Provider } = requireNapiRsModule(
      "@nomicfoundation/edr"
    ) as typeof import("@nomicfoundation/edr");

    const coinbase = config.coinbase ?? DEFAULT_COINBASE;

    let fork;
    if (config.forkConfig !== undefined) {
      let httpHeaders: HttpHeader[] | undefined;
      if (config.forkConfig.httpHeaders !== undefined) {
        httpHeaders = [];

        for (const [name, value] of Object.entries(
          config.forkConfig.httpHeaders
        )) {
          httpHeaders.push({
            name,
            value,
          });
        }
      }

      fork = {
        jsonRpcUrl: config.forkConfig.jsonRpcUrl,
        blockNumber:
          config.forkConfig.blockNumber !== undefined
            ? BigInt(config.forkConfig.blockNumber)
            : undefined,
        httpHeaders,
      };
    }

    const initialDate =
      config.initialDate !== undefined
        ? BigInt(Math.floor(config.initialDate.getTime() / 1000))
        : undefined;

    // To accommodate construction ordering, we need an adapter to forward events
    // from the EdrProvider callback to the wrapper's listener
    const eventAdapter = new EdrProviderEventAdapter();

    const printLineFn = loggerConfig.printLineFn ?? printLine;
    const replaceLastLineFn = loggerConfig.replaceLastLineFn ?? replaceLastLine;

    const hardforkName = getHardforkName(config.hardfork);

    const provider = await Provider.withConfig(
      getGlobalEdrContext(),
      {
        allowBlocksWithSameTimestamp:
          config.allowBlocksWithSameTimestamp ?? false,
        allowUnlimitedContractSize: config.allowUnlimitedContractSize,
        bailOnCallFailure: config.throwOnCallFailures,
        bailOnTransactionFailure: config.throwOnTransactionFailures,
        blockGasLimit: BigInt(config.blockGasLimit),
        chainId: BigInt(config.chainId),
        chains: Array.from(config.chains, ([chainId, hardforkConfig]) => {
          return {
            chainId: BigInt(chainId),
            hardforks: Array.from(
              hardforkConfig.hardforkHistory,
              ([hardfork, blockNumber]) => {
                return {
                  blockNumber: BigInt(blockNumber),
                  specId: ethereumsjsHardforkToEdrSpecId(
                    getHardforkName(hardfork)
                  ),
                };
              }
            ),
          };
        }),
        cacheDir: config.forkCachePath,
        coinbase: Buffer.from(coinbase.slice(2), "hex"),
        enableRip7212: config.enableRip7212,
        fork,
        hardfork: ethereumsjsHardforkToEdrSpecId(hardforkName),
        genesisAccounts: config.genesisAccounts.map((account) => {
          return {
            secretKey: account.privateKey,
            balance: BigInt(account.balance),
          };
        }),
        initialDate,
        initialBaseFeePerGas:
          config.initialBaseFeePerGas !== undefined
            ? BigInt(config.initialBaseFeePerGas!)
            : undefined,
        minGasPrice: config.minGasPrice,
        mining: {
          autoMine: config.automine,
          interval: ethereumjsIntervalMiningConfigToEdr(config.intervalMining),
          memPool: {
            order: ethereumjsMempoolOrderToEdrMineOrdering(config.mempoolOrder),
          },
        },
        networkId: BigInt(config.networkId),
      },
      {
        enable: loggerConfig.enabled,
        decodeConsoleLogInputsCallback: ConsoleLogger.getDecodedLogs,
        printLineCallback: (message: string, replace: boolean) => {
          if (replace) {
            replaceLastLineFn(message);
          } else {
            printLineFn(message);
          }
        },
      },
      tracingConfig ?? {},
      (event: SubscriptionEvent) => {
        eventAdapter.emit("ethEvent", event);
      }
    );

    const minimalEthereumJsNode = {
      _vm: getMinimalEthereumJsVm(provider),
    };

    const wrapper = new EdrProviderWrapper(provider, minimalEthereumJsNode);

    // Pass through all events from the provider
    eventAdapter.addListener(
      "ethEvent",
      wrapper._ethEventListener.bind(wrapper)
    );

    return wrapper;
  }

  public async request(args: RequestArguments): Promise<unknown> {
    if (args.params !== undefined && !Array.isArray(args.params)) {
      throw new InvalidInputError(
        "Hardhat Network doesn't support JSON-RPC params sent as an object"
      );
    }

    const params = args.params ?? [];

    if (args.method === "hardhat_getStackTraceFailuresCount") {
      // stubbed for backwards compatibility
      return 0;
    }

    const stringifiedArgs = JSON.stringify({
      method: args.method,
      params,
    });

    const responseObject: Response = await this._provider.handleRequest(
      stringifiedArgs
    );

    let response;
    if (typeof responseObject.data === "string") {
      response = JSON.parse(responseObject.data);
    } else {
      response = responseObject.data;
    }

    const needsTraces =
      this._node._vm.evm.events.eventNames().length > 0 ||
      this._node._vm.events.eventNames().length > 0;

    if (needsTraces) {
      const rawTraces = responseObject.traces;
      for (const rawTrace of rawTraces) {
        // For other consumers in JS we need to marshall the entire trace over FFI
        const trace = rawTrace.trace();

        // beforeTx event
        if (this._node._vm.events.listenerCount("beforeTx") > 0) {
          this._node._vm.events.emit("beforeTx");
        }

        for (const traceItem of trace) {
          // step event
          if ("pc" in traceItem) {
            if (this._node._vm.evm.events.listenerCount("step") > 0) {
              this._node._vm.evm.events.emit(
                "step",
                edrTracingStepToMinimalInterpreterStep(traceItem)
              );
            }
          }
          // afterMessage event
          else if ("executionResult" in traceItem) {
            if (this._node._vm.evm.events.listenerCount("afterMessage") > 0) {
              this._node._vm.evm.events.emit(
                "afterMessage",
                edrTracingMessageResultToMinimalEVMResult(traceItem)
              );
            }
          }
          // beforeMessage event
          else {
            if (this._node._vm.evm.events.listenerCount("beforeMessage") > 0) {
              this._node._vm.evm.events.emit(
                "beforeMessage",
                edrTracingMessageToMinimalMessage(traceItem)
              );
            }
          }
        }

        // afterTx event
        if (this._node._vm.events.listenerCount("afterTx") > 0) {
          this._node._vm.events.emit("afterTx");
        }
      }
    }

    if (isErrorResponse(response)) {
      let error;

      let stackTrace: SolidityStackTrace | null = null;
      try {
        stackTrace = responseObject.stackTrace();
      } catch (e) {
        log("Failed to get stack trace: %O", e);
      }

      if (stackTrace !== null) {
        error = encodeSolidityStackTrace(response.error.message, stackTrace);
        // Pass data and transaction hash from the original error
        (error as any).data = response.error.data?.data ?? undefined;
        (error as any).transactionHash =
          response.error.data?.transactionHash ?? undefined;
      } else {
        if (response.error.code === InvalidArgumentsError.CODE) {
          error = new InvalidArgumentsError(response.error.message);
        } else {
          error = new ProviderError(
            response.error.message,
            response.error.code
          );
        }
        error.data = response.error.data;
      }

      // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
      throw error;
    }

    if (args.method === "hardhat_reset") {
      this.emit(HARDHAT_NETWORK_RESET_EVENT);
    } else if (args.method === "evm_revert") {
      this.emit(HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT);
    }

    // Override EDR version string with Hardhat version string with EDR backend,
    // e.g. `HardhatNetwork/2.19.0/@nomicfoundation/edr/0.2.0-dev`
    if (args.method === "web3_clientVersion") {
      return clientVersion(response.result);
    } else if (
      args.method === "debug_traceTransaction" ||
      args.method === "debug_traceCall"
    ) {
      return edrRpcDebugTraceToHardhat(response.result);
    } else {
      return response.result;
    }
  }

  // temporarily added to make smock work with HH+EDR
  private _setCallOverrideCallback(callback: CallOverrideCallback) {
    this._callOverrideCallback = callback;

    this._provider.setCallOverrideCallback(
      async (address: Buffer, data: Buffer) => {
        return this._callOverrideCallback?.(address, data);
      }
    );
  }

  private _setVerboseTracing(enabled: boolean) {
    this._provider.setVerboseTracing(enabled);
  }

  private _ethEventListener(event: SubscriptionEvent) {
    const subscription = `0x${event.filterId.toString(16)}`;
    const results = Array.isArray(event.result) ? event.result : [event.result];
    for (const result of results) {
      this._emitLegacySubscriptionEvent(subscription, result);
      this._emitEip1193SubscriptionEvent(subscription, result);
    }
  }

  private _emitLegacySubscriptionEvent(subscription: string, result: any) {
    this.emit("notification", {
      subscription,
      result,
    });
  }

  private _emitEip1193SubscriptionEvent(subscription: string, result: unknown) {
    const message: EthSubscription = {
      type: "eth_subscription",
      data: {
        subscription,
        result,
      },
    };

    this.emit("message", message);
  }
}

async function clientVersion(edrClientVersion: string): Promise<string> {
  const hardhatPackage = await getPackageJson();
  const edrVersion = edrClientVersion.split("/")[1];
  return `HardhatNetwork/${hardhatPackage.version}/@nomicfoundation/edr/${edrVersion}`;
}

export async function createHardhatNetworkProvider(
  hardhatNetworkProviderConfig: HardhatNetworkProviderConfig,
  loggerConfig: LoggerConfig,
  artifacts?: Artifacts
): Promise<EIP1193Provider> {
  log("Making tracing config");
  const tracingConfig = await makeTracingConfig(artifacts);
  log("Creating EDR provider");
  const provider = await EdrProviderWrapper.create(
    hardhatNetworkProviderConfig,
    loggerConfig,
    tracingConfig
  );
  log("EDR provider created");

  return provider;
}

async function makeTracingConfig(
  artifacts: Artifacts | undefined
): Promise<TracingConfigWithBuffers | undefined> {
  if (artifacts !== undefined) {
    const buildInfoFiles = await artifacts.getBuildInfoPaths();

    try {
      const buildInfos = await Promise.all(
        buildInfoFiles.map((filePath) => fsExtra.readFile(filePath))
      );

      return {
        buildInfos,
      };
    } catch (error) {
      console.warn(
        picocolors.yellow(
          "Stack traces engine could not be initialized. Run Hardhat with --verbose to learn more."
        )
      );

      log(
        "Solidity stack traces disabled: Failed to read solc's input and output files. Please report this to help us improve Hardhat.\n",
        error
      );
    }
  }
}
