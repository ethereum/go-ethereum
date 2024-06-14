import { MessageTrace } from "../internal/hardhat-network/stack-traces/message-trace";
import { HardhatRuntimeEnvironment } from "./runtime";
export type ExperimentalHardhatNetworkMessageTraceHook = (hre: HardhatRuntimeEnvironment, trace: MessageTrace, isMessageTraceFromACall: boolean) => Promise<void>;
export type BoundExperimentalHardhatNetworkMessageTraceHook = (trace: MessageTrace, isMessageTraceFromACall: boolean) => Promise<void>;
//# sourceMappingURL=experimental.d.ts.map