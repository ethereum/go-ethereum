"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.edrTracingMessageToMinimalMessage = exports.edrTracingMessageResultToMinimalEVMResult = exports.edrTracingStepToMinimalInterpreterStep = exports.edrRpcDebugTraceToHardhat = exports.ethereumjsMempoolOrderToEdrMineOrdering = exports.ethereumjsIntervalMiningConfigToEdr = exports.edrSpecIdToEthereumHardfork = exports.ethereumsjsHardforkToEdrSpecId = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const napi_rs_1 = require("../../../../common/napi-rs");
const hardforks_1 = require("../../../util/hardforks");
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
function ethereumsjsHardforkToEdrSpecId(hardfork) {
    const { SpecId } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
    switch (hardfork) {
        case hardforks_1.HardforkName.FRONTIER:
            return 0 /* SpecId.Frontier */;
        case hardforks_1.HardforkName.HOMESTEAD:
            return 2 /* SpecId.Homestead */;
        case hardforks_1.HardforkName.DAO:
            return 3 /* SpecId.DaoFork */;
        case hardforks_1.HardforkName.TANGERINE_WHISTLE:
            return 4 /* SpecId.Tangerine */;
        case hardforks_1.HardforkName.SPURIOUS_DRAGON:
            return 5 /* SpecId.SpuriousDragon */;
        case hardforks_1.HardforkName.BYZANTIUM:
            return 6 /* SpecId.Byzantium */;
        case hardforks_1.HardforkName.CONSTANTINOPLE:
            return 7 /* SpecId.Constantinople */;
        case hardforks_1.HardforkName.PETERSBURG:
            return 8 /* SpecId.Petersburg */;
        case hardforks_1.HardforkName.ISTANBUL:
            return 9 /* SpecId.Istanbul */;
        case hardforks_1.HardforkName.MUIR_GLACIER:
            return 10 /* SpecId.MuirGlacier */;
        case hardforks_1.HardforkName.BERLIN:
            return 11 /* SpecId.Berlin */;
        case hardforks_1.HardforkName.LONDON:
            return 12 /* SpecId.London */;
        case hardforks_1.HardforkName.ARROW_GLACIER:
            return 13 /* SpecId.ArrowGlacier */;
        case hardforks_1.HardforkName.GRAY_GLACIER:
            return 14 /* SpecId.GrayGlacier */;
        case hardforks_1.HardforkName.MERGE:
            return 15 /* SpecId.Merge */;
        case hardforks_1.HardforkName.SHANGHAI:
            return 16 /* SpecId.Shanghai */;
        case hardforks_1.HardforkName.CANCUN:
            return 17 /* SpecId.Cancun */;
        default:
            const _exhaustiveCheck = hardfork;
            throw new Error(`Unknown hardfork name '${hardfork}', this shouldn't happen`);
    }
}
exports.ethereumsjsHardforkToEdrSpecId = ethereumsjsHardforkToEdrSpecId;
function edrSpecIdToEthereumHardfork(specId) {
    const { SpecId } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
    switch (specId) {
        case 0 /* SpecId.Frontier */:
            return hardforks_1.HardforkName.FRONTIER;
        case 2 /* SpecId.Homestead */:
            return hardforks_1.HardforkName.HOMESTEAD;
        case 3 /* SpecId.DaoFork */:
            return hardforks_1.HardforkName.DAO;
        case 4 /* SpecId.Tangerine */:
            return hardforks_1.HardforkName.TANGERINE_WHISTLE;
        case 5 /* SpecId.SpuriousDragon */:
            return hardforks_1.HardforkName.SPURIOUS_DRAGON;
        case 6 /* SpecId.Byzantium */:
            return hardforks_1.HardforkName.BYZANTIUM;
        case 7 /* SpecId.Constantinople */:
            return hardforks_1.HardforkName.CONSTANTINOPLE;
        case 8 /* SpecId.Petersburg */:
            return hardforks_1.HardforkName.PETERSBURG;
        case 9 /* SpecId.Istanbul */:
            return hardforks_1.HardforkName.ISTANBUL;
        case 10 /* SpecId.MuirGlacier */:
            return hardforks_1.HardforkName.MUIR_GLACIER;
        case 11 /* SpecId.Berlin */:
            return hardforks_1.HardforkName.BERLIN;
        case 12 /* SpecId.London */:
            return hardforks_1.HardforkName.LONDON;
        case 13 /* SpecId.ArrowGlacier */:
            return hardforks_1.HardforkName.ARROW_GLACIER;
        case 14 /* SpecId.GrayGlacier */:
            return hardforks_1.HardforkName.GRAY_GLACIER;
        case 15 /* SpecId.Merge */:
            return hardforks_1.HardforkName.MERGE;
        case 16 /* SpecId.Shanghai */:
            return hardforks_1.HardforkName.SHANGHAI;
        // HACK: EthereumJS doesn't support Cancun, so report Shanghai
        case 17 /* SpecId.Cancun */:
            return hardforks_1.HardforkName.SHANGHAI;
        default:
            throw new Error(`Unknown spec id '${specId}', this shouldn't happen`);
    }
}
exports.edrSpecIdToEthereumHardfork = edrSpecIdToEthereumHardfork;
function ethereumjsIntervalMiningConfigToEdr(config) {
    if (typeof config === "number") {
        // Is interval mining disabled?
        if (config === 0) {
            return undefined;
        }
        else {
            return BigInt(config);
        }
    }
    else {
        return {
            min: BigInt(config[0]),
            max: BigInt(config[1]),
        };
    }
}
exports.ethereumjsIntervalMiningConfigToEdr = ethereumjsIntervalMiningConfigToEdr;
function ethereumjsMempoolOrderToEdrMineOrdering(mempoolOrder) {
    const { MineOrdering } = (0, napi_rs_1.requireNapiRsModule)("@nomicfoundation/edr");
    switch (mempoolOrder) {
        case "fifo":
            return "Fifo" /* MineOrdering.Fifo */;
        case "priority":
            return "Priority" /* MineOrdering.Priority */;
    }
}
exports.ethereumjsMempoolOrderToEdrMineOrdering = ethereumjsMempoolOrderToEdrMineOrdering;
function edrRpcDebugTraceToHardhat(rpcDebugTrace) {
    const structLogs = rpcDebugTrace.structLogs.map((log) => {
        const result = {
            depth: Number(log.depth),
            gas: Number(log.gas),
            gasCost: Number(log.gasCost),
            op: log.opName,
            pc: Number(log.pc),
        };
        if (log.memory !== undefined) {
            result.memory = log.memory;
        }
        if (log.stack !== undefined) {
            // Remove 0x prefix which is required by EIP-3155, but not expected by Hardhat.
            result.stack = log.stack?.map((item) => item.slice(2));
        }
        if (log.storage !== undefined) {
            result.storage = Object.fromEntries(Object.entries(log.storage).map(([key, value]) => {
                return [key.slice(2), value.slice(2)];
            }));
        }
        if (log.error !== undefined) {
            result.error = {
                message: log.error,
            };
        }
        return result;
    });
    // REVM trace adds initial STOP that Hardhat doesn't expect
    if (structLogs.length > 0 && structLogs[0].op === "STOP") {
        structLogs.shift();
    }
    let returnValue = rpcDebugTrace.output?.toString("hex") ?? "";
    if (returnValue === "0x") {
        returnValue = "";
    }
    return {
        failed: !rpcDebugTrace.pass,
        gas: Number(rpcDebugTrace.gasUsed),
        returnValue,
        structLogs,
    };
}
exports.edrRpcDebugTraceToHardhat = edrRpcDebugTraceToHardhat;
function edrTracingStepToMinimalInterpreterStep(step) {
    const minimalInterpreterStep = {
        pc: Number(step.pc),
        depth: step.depth,
        opcode: {
            name: step.opcode,
        },
        stack: step.stack,
    };
    if (step.memory !== undefined) {
        minimalInterpreterStep.memory = step.memory;
    }
    return minimalInterpreterStep;
}
exports.edrTracingStepToMinimalInterpreterStep = edrTracingStepToMinimalInterpreterStep;
function edrTracingMessageResultToMinimalEVMResult(tracingMessageResult) {
    const { result, contractAddress } = tracingMessageResult.executionResult;
    // only SuccessResult has logs
    const success = "logs" in result;
    const minimalEVMResult = {
        execResult: {
            executionGasUsed: result.gasUsed,
            success,
        },
    };
    // only success and exceptional halt have reason
    if ("reason" in result) {
        minimalEVMResult.execResult.reason = result.reason;
    }
    if ("output" in result) {
        const { output } = result;
        if (Buffer.isBuffer(output)) {
            minimalEVMResult.execResult.output = output;
        }
        else {
            minimalEVMResult.execResult.output = output.returnValue;
        }
    }
    if (contractAddress !== undefined) {
        minimalEVMResult.execResult.contractAddress = new ethereumjs_util_1.Address(contractAddress);
    }
    return minimalEVMResult;
}
exports.edrTracingMessageResultToMinimalEVMResult = edrTracingMessageResultToMinimalEVMResult;
function edrTracingMessageToMinimalMessage(message) {
    return {
        to: message.to !== undefined ? new ethereumjs_util_1.Address(message.to) : undefined,
        codeAddress: message.codeAddress !== undefined
            ? new ethereumjs_util_1.Address(message.codeAddress)
            : undefined,
        data: message.data,
        value: message.value,
        caller: new ethereumjs_util_1.Address(message.caller),
        gasLimit: message.gasLimit,
        isStaticCall: message.isStaticCall,
    };
}
exports.edrTracingMessageToMinimalMessage = edrTracingMessageToMinimalMessage;
//# sourceMappingURL=convertToEdr.js.map