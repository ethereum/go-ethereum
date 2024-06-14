import type { SpecId, MineOrdering, IntervalRange, DebugTraceResult, TracingMessage, TracingMessageResult, TracingStep } from "@nomicfoundation/edr";
import { HardforkName } from "../../../util/hardforks";
import { IntervalMiningConfig, MempoolOrder } from "../node-types";
import { RpcDebugTraceOutput } from "../output";
import { MinimalEVMResult, MinimalInterpreterStep, MinimalMessage } from "../vm/types";
export declare function ethereumsjsHardforkToEdrSpecId(hardfork: HardforkName): SpecId;
export declare function edrSpecIdToEthereumHardfork(specId: SpecId): HardforkName;
export declare function ethereumjsIntervalMiningConfigToEdr(config: IntervalMiningConfig): bigint | IntervalRange | undefined;
export declare function ethereumjsMempoolOrderToEdrMineOrdering(mempoolOrder: MempoolOrder): MineOrdering;
export declare function edrRpcDebugTraceToHardhat(rpcDebugTrace: DebugTraceResult): RpcDebugTraceOutput;
export declare function edrTracingStepToMinimalInterpreterStep(step: TracingStep): MinimalInterpreterStep;
export declare function edrTracingMessageResultToMinimalEVMResult(tracingMessageResult: TracingMessageResult): MinimalEVMResult;
export declare function edrTracingMessageToMinimalMessage(message: TracingMessage): MinimalMessage;
//# sourceMappingURL=convertToEdr.d.ts.map