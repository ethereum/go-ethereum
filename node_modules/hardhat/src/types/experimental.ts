import { MessageTrace } from "../internal/hardhat-network/stack-traces/message-trace";

import { HardhatRuntimeEnvironment } from "./runtime";

// NOTE: This is experimental and will be removed. Please contact our team
// if you are planning to use it.
export type ExperimentalHardhatNetworkMessageTraceHook = (
  hre: HardhatRuntimeEnvironment,
  trace: MessageTrace,
  isMessageTraceFromACall: boolean
) => Promise<void>;

// NOTE: This is experimental and will be removed. Please contact our team
// if you are planning to use it.
export type BoundExperimentalHardhatNetworkMessageTraceHook = (
  trace: MessageTrace,
  isMessageTraceFromACall: boolean
) => Promise<void>;
