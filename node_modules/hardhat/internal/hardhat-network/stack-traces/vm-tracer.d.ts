import type { ExecutionResult, TracingMessage, TracingStep } from "@nomicfoundation/edr";
import { Exit } from "../provider/vm/exit";
import { MessageTrace } from "./message-trace";
export declare class VMTracer {
    private readonly _throwErrors;
    tracingSteps: TracingStep[];
    private _messageTraces;
    private _lastError;
    private _maxPrecompileNumber;
    constructor(_throwErrors?: boolean);
    getLastTopLevelMessageTrace(): MessageTrace | undefined;
    getLastError(): Error | undefined;
    clearLastError(): void;
    private _shouldKeepTracing;
    addBeforeMessage(message: TracingMessage): Promise<void>;
    addStep(step: TracingStep): Promise<void>;
    addAfterMessage(result: ExecutionResult, haltOverride?: Exit): Promise<void>;
}
//# sourceMappingURL=vm-tracer.d.ts.map