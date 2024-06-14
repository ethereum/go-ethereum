import type {
  Artifacts,
  BoundExperimentalHardhatNetworkMessageTraceHook,
  CompilerInput,
  CompilerOutput,
  EIP1193Provider,
  EthSubscription,
  HardhatNetworkChainsConfig,
  RequestArguments,
} from "../../../types";

import type {
  EdrContext,
  Provider as EdrProviderT,
  ExecutionResult,
  RawTrace,
  Response,
  SubscriptionEvent,
  TracingMessage,
  TracingStep,
} from "@nomicfoundation/edr";
import { Common } from "@nomicfoundation/ethereumjs-common";
import chalk from "chalk";
import debug from "debug";
import { EventEmitter } from "events";
import fsExtra from "fs-extra";
import * as t from "io-ts";
import semver from "semver";

import { requireNapiRsModule } from "../../../common/napi-rs";
import {
  HARDHAT_NETWORK_RESET_EVENT,
  HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT,
} from "../../constants";
import {
  rpcCompilerInput,
  rpcCompilerOutput,
} from "../../core/jsonrpc/types/input/solc";
import { validateParams } from "../../core/jsonrpc/types/input/validation";
import {
  InvalidArgumentsError,
  InvalidInputError,
  ProviderError,
} from "../../core/providers/errors";
import { isErrorResponse } from "../../core/providers/http";
import { getHardforkName } from "../../util/hardforks";
import { createModelsAndDecodeBytecodes } from "../stack-traces/compiler-to-model";
import { ConsoleLogger } from "../stack-traces/consoleLogger";
import { ContractsIdentifier } from "../stack-traces/contracts-identifier";
import {
  VmTraceDecoder,
  initializeVmTraceDecoder,
} from "../stack-traces/vm-trace-decoder";
import { FIRST_SOLC_VERSION_SUPPORTED } from "../stack-traces/constants";
import { encodeSolidityStackTrace } from "../stack-traces/solidity-errors";
import { SolidityStackTrace } from "../stack-traces/solidity-stack-trace";
import { SolidityTracer } from "../stack-traces/solidityTracer";
import { VMTracer } from "../stack-traces/vm-tracer";

import { getPackageJson } from "../../util/packageInfo";
import {
  ForkConfig,
  GenesisAccount,
  IntervalMiningConfig,
  MempoolOrder,
  NodeConfig,
  TracingConfig,
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
import { makeCommon } from "./utils/makeCommon";
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
  experimentalHardhatNetworkMessageTraceHooks?: BoundExperimentalHardhatNetworkMessageTraceHook[];
  forkConfig?: ForkConfig;
  forkCachePath?: string;
  enableTransientStorage: boolean;
}

export function getNodeConfig(
  config: HardhatNetworkProviderConfig,
  tracingConfig?: TracingConfig
): NodeConfig {
  return {
    automine: config.automine,
    blockGasLimit: config.blockGasLimit,
    minGasPrice: config.minGasPrice,
    genesisAccounts: config.genesisAccounts,
    allowUnlimitedContractSize: config.allowUnlimitedContractSize,
    tracingConfig,
    initialBaseFeePerGas: config.initialBaseFeePerGas,
    mempoolOrder: config.mempoolOrder,
    hardfork: config.hardfork,
    chainId: config.chainId,
    networkId: config.networkId,
    initialDate: config.initialDate,
    forkConfig: config.forkConfig,
    forkCachePath:
      config.forkConfig !== undefined ? config.forkCachePath : undefined,
    coinbase: config.coinbase ?? DEFAULT_COINBASE,
    chains: config.chains,
    allowBlocksWithSameTimestamp: config.allowBlocksWithSameTimestamp,
    enableTransientStorage: config.enableTransientStorage,
  };
}

export interface RawTraceCallbacks {
  onStep?: (messageTrace: TracingStep) => Promise<void>;
  onBeforeMessage?: (messageTrace: TracingMessage) => Promise<void>;
  onAfterMessage?: (messageTrace: ExecutionResult) => Promise<void>;
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
    },
    private readonly _eventAdapter: EdrProviderEventAdapter,
    private readonly _vmTraceDecoder: VmTraceDecoder,
    private readonly _rawTraceCallbacks: RawTraceCallbacks,
    // The common configuration for EthereumJS VM is not used by EDR, but tests expect it as part of the provider.
    private readonly _common: Common,
    tracingConfig?: TracingConfig
  ) {
    super();

    if (tracingConfig !== undefined) {
      initializeVmTraceDecoder(this._vmTraceDecoder, tracingConfig);
    }
  }

  public static async create(
    config: HardhatNetworkProviderConfig,
    loggerConfig: LoggerConfig,
    rawTraceCallbacks: RawTraceCallbacks,
    tracingConfig?: TracingConfig
  ): Promise<EdrProviderWrapper> {
    const { Provider } = requireNapiRsModule(
      "@nomicfoundation/edr"
    ) as typeof import("@nomicfoundation/edr");

    const coinbase = config.coinbase ?? DEFAULT_COINBASE;

    let fork;
    if (config.forkConfig !== undefined) {
      fork = {
        jsonRpcUrl: config.forkConfig.jsonRpcUrl,
        blockNumber:
          config.forkConfig.blockNumber !== undefined
            ? BigInt(config.forkConfig.blockNumber)
            : undefined,
      };
    }

    const initialDate =
      config.initialDate !== undefined
        ? BigInt(Math.floor(config.initialDate.getTime() / 1000))
        : undefined;

    // To accomodate construction ordering, we need an adapter to forward events
    // from the EdrProvider callback to the wrapper's listener
    const eventAdapter = new EdrProviderEventAdapter();

    const printLineFn = loggerConfig.printLineFn ?? printLine;
    const replaceLastLineFn = loggerConfig.replaceLastLineFn ?? replaceLastLine;

    const contractsIdentifier = new ContractsIdentifier();
    const vmTraceDecoder = new VmTraceDecoder(contractsIdentifier);

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
        decodeConsoleLogInputsCallback: (inputs: Buffer[]) => {
          const consoleLogger = new ConsoleLogger();
          return consoleLogger.getDecodedLogs(inputs);
        },
        getContractAndFunctionNameCallback: (
          code: Buffer,
          calldata?: Buffer
        ) => {
          return vmTraceDecoder.getContractAndFunctionNamesForCall(
            code,
            calldata
          );
        },
        printLineCallback: (message: string, replace: boolean) => {
          if (replace) {
            replaceLastLineFn(message);
          } else {
            printLineFn(message);
          }
        },
      },
      (event: SubscriptionEvent) => {
        eventAdapter.emit("ethEvent", event);
      }
    );

    const minimalEthereumJsNode = {
      _vm: getMinimalEthereumJsVm(provider),
    };

    const common = makeCommon(getNodeConfig(config));
    const wrapper = new EdrProviderWrapper(
      provider,
      minimalEthereumJsNode,
      eventAdapter,
      vmTraceDecoder,
      rawTraceCallbacks,
      common,
      tracingConfig
    );

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

    if (args.method === "hardhat_addCompilationResult") {
      return this._addCompilationResultAction(
        ...this._addCompilationResultParams(params)
      );
    } else if (args.method === "hardhat_getStackTraceFailuresCount") {
      return this._getStackTraceFailuresCountAction(
        ...this._getStackTraceFailuresCountParams(params)
      );
    }

    const stringifiedArgs = JSON.stringify({
      method: args.method,
      params,
    });

    const responseObject: Response = await this._provider.handleRequest(
      stringifiedArgs
    );
    const response = JSON.parse(responseObject.json);

    const needsTraces =
      this._node._vm.evm.events.eventNames().length > 0 ||
      this._node._vm.events.eventNames().length > 0 ||
      this._rawTraceCallbacks.onStep !== undefined ||
      this._rawTraceCallbacks.onAfterMessage !== undefined ||
      this._rawTraceCallbacks.onBeforeMessage !== undefined;

    if (needsTraces) {
      const rawTraces = responseObject.traces;
      for (const rawTrace of rawTraces) {
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
            if (this._rawTraceCallbacks.onStep !== undefined) {
              await this._rawTraceCallbacks.onStep(traceItem);
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
            if (this._rawTraceCallbacks.onAfterMessage !== undefined) {
              await this._rawTraceCallbacks.onAfterMessage(
                traceItem.executionResult
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
            if (this._rawTraceCallbacks.onBeforeMessage !== undefined) {
              await this._rawTraceCallbacks.onBeforeMessage(traceItem);
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

      const solidityTrace = responseObject.solidityTrace;
      let stackTrace: SolidityStackTrace | undefined;
      if (solidityTrace !== null) {
        stackTrace = await this._rawTraceToSolidityStackTrace(solidityTrace);
      }

      if (stackTrace !== undefined) {
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

  private _addCompilationResultParams(
    params: any[]
  ): [string, CompilerInput, CompilerOutput] {
    return validateParams(
      params,
      t.string,
      rpcCompilerInput,
      rpcCompilerOutput
    );
  }

  private async _addCompilationResultAction(
    solcVersion: string,
    compilerInput: CompilerInput,
    compilerOutput: CompilerOutput
  ): Promise<boolean> {
    let bytecodes;
    try {
      bytecodes = createModelsAndDecodeBytecodes(
        solcVersion,
        compilerInput,
        compilerOutput
      );
    } catch (error) {
      console.warn(
        chalk.yellow(
          "The Hardhat Network tracing engine could not be updated. Run Hardhat with --verbose to learn more."
        )
      );

      log(
        "ContractsIdentifier failed to be updated. Please report this to help us improve Hardhat.\n",
        error
      );

      return false;
    }

    for (const bytecode of bytecodes) {
      this._vmTraceDecoder.addBytecode(bytecode);
    }

    return true;
  }

  private _getStackTraceFailuresCountParams(params: any[]): [] {
    return validateParams(params);
  }

  private _getStackTraceFailuresCountAction(): number {
    return this._failedStackTraces;
  }

  private async _rawTraceToSolidityStackTrace(
    rawTrace: RawTrace
  ): Promise<SolidityStackTrace | undefined> {
    const vmTracer = new VMTracer(false);

    const trace = rawTrace.trace();
    for (const traceItem of trace) {
      if ("pc" in traceItem) {
        await vmTracer.addStep(traceItem);
      } else if ("executionResult" in traceItem) {
        await vmTracer.addAfterMessage(traceItem.executionResult);
      } else {
        await vmTracer.addBeforeMessage(traceItem);
      }
    }

    let vmTrace = vmTracer.getLastTopLevelMessageTrace();
    const vmTracerError = vmTracer.getLastError();

    if (vmTrace !== undefined) {
      vmTrace = this._vmTraceDecoder.tryToDecodeMessageTrace(vmTrace);
    }

    try {
      if (vmTrace === undefined || vmTracerError !== undefined) {
        throw vmTracerError;
      }

      const solidityTracer = new SolidityTracer();
      return solidityTracer.getStackTrace(vmTrace);
    } catch (err) {
      this._failedStackTraces += 1;
      log(
        "Could not generate stack trace. Please report this to help us improve Hardhat.\n",
        err
      );
    }
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
  return EdrProviderWrapper.create(
    hardhatNetworkProviderConfig,
    loggerConfig,
    {},
    await makeTracingConfig(artifacts)
  );
}

async function makeTracingConfig(
  artifacts: Artifacts | undefined
): Promise<TracingConfig | undefined> {
  if (artifacts !== undefined) {
    const buildInfos = [];

    const buildInfoFiles = await artifacts.getBuildInfoPaths();

    try {
      for (const buildInfoFile of buildInfoFiles) {
        const buildInfo = await fsExtra.readJson(buildInfoFile);
        if (semver.gte(buildInfo.solcVersion, FIRST_SOLC_VERSION_SUPPORTED)) {
          buildInfos.push(buildInfo);
        }
      }

      return {
        buildInfos,
      };
    } catch (error) {
      console.warn(
        chalk.yellow(
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
