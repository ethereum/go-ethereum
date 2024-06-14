"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.createHardhatNetworkProvider = exports.EdrProviderWrapper = exports.getNodeConfig = exports.getGlobalEdrContext = exports.DEFAULT_COINBASE = void 0;
const chalk_1 = __importDefault(require("chalk"));
const debug_1 = __importDefault(require("debug"));
const events_1 = require("events");
const fs_extra_1 = __importDefault(require("fs-extra"));
const t = __importStar(require("io-ts"));
const semver_1 = __importDefault(require("semver"));
const napi_rs_1 = require("../../../common/napi-rs");
const constants_1 = require("../../constants");
const solc_1 = require("../../core/jsonrpc/types/input/solc");
const validation_1 = require("../../core/jsonrpc/types/input/validation");
const errors_1 = require("../../core/providers/errors");
const http_1 = require("../../core/providers/http");
const hardforks_1 = require("../../util/hardforks");
const compiler_to_model_1 = require("../stack-traces/compiler-to-model");
const consoleLogger_1 = require("../stack-traces/consoleLogger");
const contracts_identifier_1 = require("../stack-traces/contracts-identifier");
const vm_trace_decoder_1 = require("../stack-traces/vm-trace-decoder");
const constants_2 = require("../stack-traces/constants");
const solidity_errors_1 = require("../stack-traces/solidity-errors");
const solidityTracer_1 = require("../stack-traces/solidityTracer");
const vm_tracer_1 = require("../stack-traces/vm-tracer");
const packageInfo_1 = require("../../util/packageInfo");
const convertToEdr_1 = require("./utils/convertToEdr");
const makeCommon_1 = require("./utils/makeCommon");
const logger_1 = require("./modules/logger");
const minimal_vm_1 = require("./vm/minimal-vm");
const log = (0, debug_1.default)("hardhat:core:hardhat-network:provider");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
exports.DEFAULT_COINBASE = "0xc014ba5ec014ba5ec014ba5ec014ba5ec014ba5e";
let _globalEdrContext;
// Lazy initialize the global EDR context.
function getGlobalEdrContext() {
    const { EdrContext } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
    if (_globalEdrContext === undefined) {
        // Only one is allowed to exist
        _globalEdrContext = new EdrContext();
    }
    return _globalEdrContext;
}
exports.getGlobalEdrContext = getGlobalEdrContext;
function getNodeConfig(config, tracingConfig) {
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
        forkCachePath: config.forkConfig !== undefined ? config.forkCachePath : undefined,
        coinbase: config.coinbase ?? exports.DEFAULT_COINBASE,
        chains: config.chains,
        allowBlocksWithSameTimestamp: config.allowBlocksWithSameTimestamp,
        enableTransientStorage: config.enableTransientStorage,
    };
}
exports.getNodeConfig = getNodeConfig;
class EdrProviderEventAdapter extends events_1.EventEmitter {
}
class EdrProviderWrapper extends events_1.EventEmitter {
    constructor(_provider, 
    // we add this for backwards-compatibility with plugins like solidity-coverage
    _node, _eventAdapter, _vmTraceDecoder, _rawTraceCallbacks, 
    // The common configuration for EthereumJS VM is not used by EDR, but tests expect it as part of the provider.
    _common, tracingConfig) {
        super();
        this._provider = _provider;
        this._node = _node;
        this._eventAdapter = _eventAdapter;
        this._vmTraceDecoder = _vmTraceDecoder;
        this._rawTraceCallbacks = _rawTraceCallbacks;
        this._common = _common;
        this._failedStackTraces = 0;
        if (tracingConfig !== undefined) {
            (0, vm_trace_decoder_1.initializeVmTraceDecoder)(this._vmTraceDecoder, tracingConfig);
        }
    }
    static async create(config, loggerConfig, rawTraceCallbacks, tracingConfig) {
        const { Provider } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
        const coinbase = config.coinbase ?? exports.DEFAULT_COINBASE;
        let fork;
        if (config.forkConfig !== undefined) {
            fork = {
                jsonRpcUrl: config.forkConfig.jsonRpcUrl,
                blockNumber: config.forkConfig.blockNumber !== undefined
                    ? BigInt(config.forkConfig.blockNumber)
                    : undefined,
            };
        }
        const initialDate = config.initialDate !== undefined
            ? BigInt(Math.floor(config.initialDate.getTime() / 1000))
            : undefined;
        // To accomodate construction ordering, we need an adapter to forward events
        // from the EdrProvider callback to the wrapper's listener
        const eventAdapter = new EdrProviderEventAdapter();
        const printLineFn = loggerConfig.printLineFn ?? logger_1.printLine;
        const replaceLastLineFn = loggerConfig.replaceLastLineFn ?? logger_1.replaceLastLine;
        const contractsIdentifier = new contracts_identifier_1.ContractsIdentifier();
        const vmTraceDecoder = new vm_trace_decoder_1.VmTraceDecoder(contractsIdentifier);
        const hardforkName = (0, hardforks_1.getHardforkName)(config.hardfork);
        const provider = await Provider.withConfig(getGlobalEdrContext(), {
            allowBlocksWithSameTimestamp: config.allowBlocksWithSameTimestamp ?? false,
            allowUnlimitedContractSize: config.allowUnlimitedContractSize,
            bailOnCallFailure: config.throwOnCallFailures,
            bailOnTransactionFailure: config.throwOnTransactionFailures,
            blockGasLimit: BigInt(config.blockGasLimit),
            chainId: BigInt(config.chainId),
            chains: Array.from(config.chains, ([chainId, hardforkConfig]) => {
                return {
                    chainId: BigInt(chainId),
                    hardforks: Array.from(hardforkConfig.hardforkHistory, ([hardfork, blockNumber]) => {
                        return {
                            blockNumber: BigInt(blockNumber),
                            specId: (0, convertToEdr_1.ethereumsjsHardforkToEdrSpecId)((0, hardforks_1.getHardforkName)(hardfork)),
                        };
                    }),
                };
            }),
            cacheDir: config.forkCachePath,
            coinbase: Buffer.from(coinbase.slice(2), "hex"),
            fork,
            hardfork: (0, convertToEdr_1.ethereumsjsHardforkToEdrSpecId)(hardforkName),
            genesisAccounts: config.genesisAccounts.map((account) => {
                return {
                    secretKey: account.privateKey,
                    balance: BigInt(account.balance),
                };
            }),
            initialDate,
            initialBaseFeePerGas: config.initialBaseFeePerGas !== undefined
                ? BigInt(config.initialBaseFeePerGas)
                : undefined,
            minGasPrice: config.minGasPrice,
            mining: {
                autoMine: config.automine,
                interval: (0, convertToEdr_1.ethereumjsIntervalMiningConfigToEdr)(config.intervalMining),
                memPool: {
                    order: (0, convertToEdr_1.ethereumjsMempoolOrderToEdrMineOrdering)(config.mempoolOrder),
                },
            },
            networkId: BigInt(config.networkId),
        }, {
            enable: loggerConfig.enabled,
            decodeConsoleLogInputsCallback: (inputs) => {
                const consoleLogger = new consoleLogger_1.ConsoleLogger();
                return consoleLogger.getDecodedLogs(inputs);
            },
            getContractAndFunctionNameCallback: (code, calldata) => {
                return vmTraceDecoder.getContractAndFunctionNamesForCall(code, calldata);
            },
            printLineCallback: (message, replace) => {
                if (replace) {
                    replaceLastLineFn(message);
                }
                else {
                    printLineFn(message);
                }
            },
        }, (event) => {
            eventAdapter.emit("ethEvent", event);
        });
        const minimalEthereumJsNode = {
            _vm: (0, minimal_vm_1.getMinimalEthereumJsVm)(provider),
        };
        const common = (0, makeCommon_1.makeCommon)(getNodeConfig(config));
        const wrapper = new EdrProviderWrapper(provider, minimalEthereumJsNode, eventAdapter, vmTraceDecoder, rawTraceCallbacks, common, tracingConfig);
        // Pass through all events from the provider
        eventAdapter.addListener("ethEvent", wrapper._ethEventListener.bind(wrapper));
        return wrapper;
    }
    async request(args) {
        if (args.params !== undefined && !Array.isArray(args.params)) {
            throw new errors_1.InvalidInputError("Hardhat Network doesn't support JSON-RPC params sent as an object");
        }
        const params = args.params ?? [];
        if (args.method === "hardhat_addCompilationResult") {
            return this._addCompilationResultAction(...this._addCompilationResultParams(params));
        }
        else if (args.method === "hardhat_getStackTraceFailuresCount") {
            return this._getStackTraceFailuresCountAction(...this._getStackTraceFailuresCountParams(params));
        }
        const stringifiedArgs = JSON.stringify({
            method: args.method,
            params,
        });
        const responseObject = await this._provider.handleRequest(stringifiedArgs);
        const response = JSON.parse(responseObject.json);
        const needsTraces = this._node._vm.evm.events.eventNames().length > 0 ||
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
                            this._node._vm.evm.events.emit("step", (0, convertToEdr_1.edrTracingStepToMinimalInterpreterStep)(traceItem));
                        }
                        if (this._rawTraceCallbacks.onStep !== undefined) {
                            await this._rawTraceCallbacks.onStep(traceItem);
                        }
                    }
                    // afterMessage event
                    else if ("executionResult" in traceItem) {
                        if (this._node._vm.evm.events.listenerCount("afterMessage") > 0) {
                            this._node._vm.evm.events.emit("afterMessage", (0, convertToEdr_1.edrTracingMessageResultToMinimalEVMResult)(traceItem));
                        }
                        if (this._rawTraceCallbacks.onAfterMessage !== undefined) {
                            await this._rawTraceCallbacks.onAfterMessage(traceItem.executionResult);
                        }
                    }
                    // beforeMessage event
                    else {
                        if (this._node._vm.evm.events.listenerCount("beforeMessage") > 0) {
                            this._node._vm.evm.events.emit("beforeMessage", (0, convertToEdr_1.edrTracingMessageToMinimalMessage)(traceItem));
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
        if ((0, http_1.isErrorResponse)(response)) {
            let error;
            const solidityTrace = responseObject.solidityTrace;
            let stackTrace;
            if (solidityTrace !== null) {
                stackTrace = await this._rawTraceToSolidityStackTrace(solidityTrace);
            }
            if (stackTrace !== undefined) {
                error = (0, solidity_errors_1.encodeSolidityStackTrace)(response.error.message, stackTrace);
                // Pass data and transaction hash from the original error
                error.data = response.error.data?.data ?? undefined;
                error.transactionHash =
                    response.error.data?.transactionHash ?? undefined;
            }
            else {
                if (response.error.code === errors_1.InvalidArgumentsError.CODE) {
                    error = new errors_1.InvalidArgumentsError(response.error.message);
                }
                else {
                    error = new errors_1.ProviderError(response.error.message, response.error.code);
                }
                error.data = response.error.data;
            }
            // eslint-disable-next-line @nomicfoundation/hardhat-internal-rules/only-hardhat-error
            throw error;
        }
        if (args.method === "hardhat_reset") {
            this.emit(constants_1.HARDHAT_NETWORK_RESET_EVENT);
        }
        else if (args.method === "evm_revert") {
            this.emit(constants_1.HARDHAT_NETWORK_REVERT_SNAPSHOT_EVENT);
        }
        // Override EDR version string with Hardhat version string with EDR backend,
        // e.g. `HardhatNetwork/2.19.0/@nomicfoundation/edr/0.2.0-dev`
        if (args.method === "web3_clientVersion") {
            return clientVersion(response.result);
        }
        else if (args.method === "debug_traceTransaction" ||
            args.method === "debug_traceCall") {
            return (0, convertToEdr_1.edrRpcDebugTraceToHardhat)(response.result);
        }
        else {
            return response.result;
        }
    }
    // temporarily added to make smock work with HH+EDR
    _setCallOverrideCallback(callback) {
        this._callOverrideCallback = callback;
        this._provider.setCallOverrideCallback(async (address, data) => {
            return this._callOverrideCallback?.(address, data);
        });
    }
    _setVerboseTracing(enabled) {
        this._provider.setVerboseTracing(enabled);
    }
    _ethEventListener(event) {
        const subscription = `0x${event.filterId.toString(16)}`;
        const results = Array.isArray(event.result) ? event.result : [event.result];
        for (const result of results) {
            this._emitLegacySubscriptionEvent(subscription, result);
            this._emitEip1193SubscriptionEvent(subscription, result);
        }
    }
    _emitLegacySubscriptionEvent(subscription, result) {
        this.emit("notification", {
            subscription,
            result,
        });
    }
    _emitEip1193SubscriptionEvent(subscription, result) {
        const message = {
            type: "eth_subscription",
            data: {
                subscription,
                result,
            },
        };
        this.emit("message", message);
    }
    _addCompilationResultParams(params) {
        return (0, validation_1.validateParams)(params, t.string, solc_1.rpcCompilerInput, solc_1.rpcCompilerOutput);
    }
    async _addCompilationResultAction(solcVersion, compilerInput, compilerOutput) {
        let bytecodes;
        try {
            bytecodes = (0, compiler_to_model_1.createModelsAndDecodeBytecodes)(solcVersion, compilerInput, compilerOutput);
        }
        catch (error) {
            console.warn(chalk_1.default.yellow("The Hardhat Network tracing engine could not be updated. Run Hardhat with --verbose to learn more."));
            log("ContractsIdentifier failed to be updated. Please report this to help us improve Hardhat.\n", error);
            return false;
        }
        for (const bytecode of bytecodes) {
            this._vmTraceDecoder.addBytecode(bytecode);
        }
        return true;
    }
    _getStackTraceFailuresCountParams(params) {
        return (0, validation_1.validateParams)(params);
    }
    _getStackTraceFailuresCountAction() {
        return this._failedStackTraces;
    }
    async _rawTraceToSolidityStackTrace(rawTrace) {
        const vmTracer = new vm_tracer_1.VMTracer(false);
        const trace = rawTrace.trace();
        for (const traceItem of trace) {
            if ("pc" in traceItem) {
                await vmTracer.addStep(traceItem);
            }
            else if ("executionResult" in traceItem) {
                await vmTracer.addAfterMessage(traceItem.executionResult);
            }
            else {
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
            const solidityTracer = new solidityTracer_1.SolidityTracer();
            return solidityTracer.getStackTrace(vmTrace);
        }
        catch (err) {
            this._failedStackTraces += 1;
            log("Could not generate stack trace. Please report this to help us improve Hardhat.\n", err);
        }
    }
}
exports.EdrProviderWrapper = EdrProviderWrapper;
async function clientVersion(edrClientVersion) {
    const hardhatPackage = await (0, packageInfo_1.getPackageJson)();
    const edrVersion = edrClientVersion.split("/")[1];
    return `HardhatNetwork/${hardhatPackage.version}/@nomicfoundation/edr/${edrVersion}`;
}
async function createHardhatNetworkProvider(hardhatNetworkProviderConfig, loggerConfig, artifacts) {
    return EdrProviderWrapper.create(hardhatNetworkProviderConfig, loggerConfig, {}, await makeTracingConfig(artifacts));
}
exports.createHardhatNetworkProvider = createHardhatNetworkProvider;
async function makeTracingConfig(artifacts) {
    if (artifacts !== undefined) {
        const buildInfos = [];
        const buildInfoFiles = await artifacts.getBuildInfoPaths();
        try {
            for (const buildInfoFile of buildInfoFiles) {
                const buildInfo = await fs_extra_1.default.readJson(buildInfoFile);
                if (semver_1.default.gte(buildInfo.solcVersion, constants_2.FIRST_SOLC_VERSION_SUPPORTED)) {
                    buildInfos.push(buildInfo);
                }
            }
            return {
                buildInfos,
            };
        }
        catch (error) {
            console.warn(chalk_1.default.yellow("Stack traces engine could not be initialized. Run Hardhat with --verbose to learn more."));
            log("Solidity stack traces disabled: Failed to read solc's input and output files. Please report this to help us improve Hardhat.\n", error);
        }
    }
}
//# sourceMappingURL=provider.js.map