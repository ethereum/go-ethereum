"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.initializeVmTraceDecoder = exports.VmTraceDecoder = void 0;
const chalk_1 = __importDefault(require("chalk"));
const debug_1 = __importDefault(require("debug"));
const reporter_1 = require("../../sentry/reporter");
const compiler_to_model_1 = require("./compiler-to-model");
const message_trace_1 = require("./message-trace");
const model_1 = require("./model");
const solidity_stack_trace_1 = require("./solidity-stack-trace");
const log = (0, debug_1.default)("hardhat:core:hardhat-network:node");
class VmTraceDecoder {
    constructor(_contractsIdentifier) {
        this._contractsIdentifier = _contractsIdentifier;
    }
    getContractAndFunctionNamesForCall(code, calldata) {
        const isCreate = calldata === undefined;
        const bytecode = this._contractsIdentifier.getBytecodeForCall(code, isCreate);
        const contractName = bytecode?.contract.name ?? solidity_stack_trace_1.UNRECOGNIZED_CONTRACT_NAME;
        if (isCreate) {
            return {
                contractName,
            };
        }
        else {
            if (bytecode === undefined) {
                return {
                    contractName,
                    functionName: "",
                };
            }
            else {
                const func = bytecode.contract.getFunctionFromSelector(calldata.slice(0, 4));
                const functionName = func === undefined
                    ? solidity_stack_trace_1.UNRECOGNIZED_FUNCTION_NAME
                    : func.type === model_1.ContractFunctionType.FALLBACK
                        ? solidity_stack_trace_1.FALLBACK_FUNCTION_NAME
                        : func.type === model_1.ContractFunctionType.RECEIVE
                            ? solidity_stack_trace_1.RECEIVE_FUNCTION_NAME
                            : func.name;
                return {
                    contractName,
                    functionName,
                };
            }
        }
    }
    tryToDecodeMessageTrace(messageTrace) {
        if ((0, message_trace_1.isPrecompileTrace)(messageTrace)) {
            return messageTrace;
        }
        return {
            ...messageTrace,
            bytecode: this._contractsIdentifier.getBytecodeForCall(messageTrace.code, (0, message_trace_1.isCreateTrace)(messageTrace)),
            steps: messageTrace.steps.map((s) => (0, message_trace_1.isEvmStep)(s) ? s : this.tryToDecodeMessageTrace(s)),
        };
    }
    addBytecode(bytecode) {
        this._contractsIdentifier.addBytecode(bytecode);
    }
}
exports.VmTraceDecoder = VmTraceDecoder;
function initializeVmTraceDecoder(vmTraceDecoder, tracingConfig) {
    if (tracingConfig.buildInfos === undefined) {
        return;
    }
    try {
        for (const buildInfo of tracingConfig.buildInfos) {
            const bytecodes = (0, compiler_to_model_1.createModelsAndDecodeBytecodes)(buildInfo.solcVersion, buildInfo.input, buildInfo.output);
            for (const bytecode of bytecodes) {
                if (tracingConfig.ignoreContracts === true &&
                    bytecode.contract.name.startsWith("Ignored")) {
                    continue;
                }
                vmTraceDecoder.addBytecode(bytecode);
            }
        }
    }
    catch (error) {
        console.warn(chalk_1.default.yellow("The Hardhat Network tracing engine could not be initialized. Run Hardhat with --verbose to learn more."));
        log("Hardhat Network tracing disabled: ContractsIdentifier failed to be initialized. Please report this to help us improve Hardhat.\n", error);
        if (error instanceof Error) {
            reporter_1.Reporter.reportError(error);
        }
    }
}
exports.initializeVmTraceDecoder = initializeVmTraceDecoder;
//# sourceMappingURL=vm-trace-decoder.js.map