import type {
  SpecId,
  MineOrdering,
  IntervalRange,
  DebugTraceResult,
  TracingMessage,
  TracingMessageResult,
  TracingStep,
} from "@nomicfoundation/edr";
import { Address } from "@nomicfoundation/ethereumjs-util";

import { requireNapiRsModule } from "../../../../common/napi-rs";
import { HardforkName } from "../../../util/hardforks";
import { IntervalMiningConfig, MempoolOrder } from "../node-types";
import { RpcDebugTraceOutput, RpcStructLog } from "../output";
import {
  MinimalEVMResult,
  MinimalInterpreterStep,
  MinimalMessage,
} from "../vm/types";

/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */

export function ethereumsjsHardforkToEdrSpecId(hardfork: HardforkName): SpecId {
  const { SpecId } = requireNapiRsModule(
    "@nomicfoundation/edr"
  ) as typeof import("@nomicfoundation/edr");

  switch (hardfork) {
    case HardforkName.FRONTIER:
      return SpecId.Frontier;
    case HardforkName.HOMESTEAD:
      return SpecId.Homestead;
    case HardforkName.DAO:
      return SpecId.DaoFork;
    case HardforkName.TANGERINE_WHISTLE:
      return SpecId.Tangerine;
    case HardforkName.SPURIOUS_DRAGON:
      return SpecId.SpuriousDragon;
    case HardforkName.BYZANTIUM:
      return SpecId.Byzantium;
    case HardforkName.CONSTANTINOPLE:
      return SpecId.Constantinople;
    case HardforkName.PETERSBURG:
      return SpecId.Petersburg;
    case HardforkName.ISTANBUL:
      return SpecId.Istanbul;
    case HardforkName.MUIR_GLACIER:
      return SpecId.MuirGlacier;
    case HardforkName.BERLIN:
      return SpecId.Berlin;
    case HardforkName.LONDON:
      return SpecId.London;
    case HardforkName.ARROW_GLACIER:
      return SpecId.ArrowGlacier;
    case HardforkName.GRAY_GLACIER:
      return SpecId.GrayGlacier;
    case HardforkName.MERGE:
      return SpecId.Merge;
    case HardforkName.SHANGHAI:
      return SpecId.Shanghai;
    case HardforkName.CANCUN:
      return SpecId.Cancun;
    default:
      const _exhaustiveCheck: never = hardfork;
      throw new Error(
        `Unknown hardfork name '${hardfork as string}', this shouldn't happen`
      );
  }
}

export function edrSpecIdToEthereumHardfork(specId: SpecId): HardforkName {
  const { SpecId } = requireNapiRsModule(
    "@nomicfoundation/edr"
  ) as typeof import("@nomicfoundation/edr");

  switch (specId) {
    case SpecId.Frontier:
      return HardforkName.FRONTIER;
    case SpecId.Homestead:
      return HardforkName.HOMESTEAD;
    case SpecId.DaoFork:
      return HardforkName.DAO;
    case SpecId.Tangerine:
      return HardforkName.TANGERINE_WHISTLE;
    case SpecId.SpuriousDragon:
      return HardforkName.SPURIOUS_DRAGON;
    case SpecId.Byzantium:
      return HardforkName.BYZANTIUM;
    case SpecId.Constantinople:
      return HardforkName.CONSTANTINOPLE;
    case SpecId.Petersburg:
      return HardforkName.PETERSBURG;
    case SpecId.Istanbul:
      return HardforkName.ISTANBUL;
    case SpecId.MuirGlacier:
      return HardforkName.MUIR_GLACIER;
    case SpecId.Berlin:
      return HardforkName.BERLIN;
    case SpecId.London:
      return HardforkName.LONDON;
    case SpecId.ArrowGlacier:
      return HardforkName.ARROW_GLACIER;
    case SpecId.GrayGlacier:
      return HardforkName.GRAY_GLACIER;
    case SpecId.Merge:
      return HardforkName.MERGE;
    case SpecId.Shanghai:
      return HardforkName.SHANGHAI;
    // HACK: EthereumJS doesn't support Cancun, so report Shanghai
    case SpecId.Cancun:
      return HardforkName.SHANGHAI;

    default:
      throw new Error(`Unknown spec id '${specId}', this shouldn't happen`);
  }
}

export function ethereumjsIntervalMiningConfigToEdr(
  config: IntervalMiningConfig
): bigint | IntervalRange | undefined {
  if (typeof config === "number") {
    // Is interval mining disabled?
    if (config === 0) {
      return undefined;
    } else {
      return BigInt(config);
    }
  } else {
    return {
      min: BigInt(config[0]),
      max: BigInt(config[1]),
    };
  }
}

export function ethereumjsMempoolOrderToEdrMineOrdering(
  mempoolOrder: MempoolOrder
): MineOrdering {
  const { MineOrdering } = requireNapiRsModule(
    "@nomicfoundation/edr"
  ) as typeof import("@nomicfoundation/edr");

  switch (mempoolOrder) {
    case "fifo":
      return MineOrdering.Fifo;
    case "priority":
      return MineOrdering.Priority;
  }
}

export function edrRpcDebugTraceToHardhat(
  rpcDebugTrace: DebugTraceResult
): RpcDebugTraceOutput {
  const structLogs = rpcDebugTrace.structLogs.map((log) => {
    const result: RpcStructLog = {
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
      result.storage = Object.fromEntries(
        Object.entries(log.storage).map(([key, value]) => {
          return [key.slice(2), value.slice(2)];
        })
      );
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

export function edrTracingStepToMinimalInterpreterStep(
  step: TracingStep
): MinimalInterpreterStep {
  const minimalInterpreterStep: MinimalInterpreterStep = {
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

export function edrTracingMessageResultToMinimalEVMResult(
  tracingMessageResult: TracingMessageResult
): MinimalEVMResult {
  const { result, contractAddress } = tracingMessageResult.executionResult;

  // only SuccessResult has logs
  const success = "logs" in result;

  const minimalEVMResult: MinimalEVMResult = {
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
    } else {
      minimalEVMResult.execResult.output = output.returnValue;
    }
  }

  if (contractAddress !== undefined) {
    minimalEVMResult.execResult.contractAddress = new Address(contractAddress);
  }

  return minimalEVMResult;
}

export function edrTracingMessageToMinimalMessage(
  message: TracingMessage
): MinimalMessage {
  return {
    to: message.to !== undefined ? new Address(message.to) : undefined,
    codeAddress:
      message.codeAddress !== undefined
        ? new Address(message.codeAddress)
        : undefined,
    data: message.data,
    value: message.value,
    caller: new Address(message.caller),
    gasLimit: message.gasLimit,
    isStaticCall: message.isStaticCall,
  };
}
