import { CallMessageTrace, CreateMessageTrace, MessageTrace, PrecompileMessageTrace } from "./message-trace";
import { SolidityStackTrace } from "./solidity-stack-trace";
export declare function printMessageTrace(trace: MessageTrace, depth?: number): void;
export declare function printCreateTrace(trace: CreateMessageTrace, depth: number): void;
export declare function printPrecompileTrace(trace: PrecompileMessageTrace, depth: number): void;
export declare function printCallTrace(trace: CallMessageTrace, depth: number): void;
export declare function printStackTrace(trace: SolidityStackTrace): void;
//# sourceMappingURL=debug.d.ts.map