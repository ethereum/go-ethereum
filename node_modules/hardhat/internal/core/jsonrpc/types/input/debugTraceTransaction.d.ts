import * as t from "io-ts";
export declare const rpcDebugTracingConfig: t.Type<{
    tracer: string | undefined;
    disableStorage: boolean | undefined;
    disableMemory: boolean | undefined;
    disableStack: boolean | undefined;
} | undefined, {
    tracer: string | undefined;
    disableStorage: boolean | undefined;
    disableMemory: boolean | undefined;
    disableStack: boolean | undefined;
} | undefined, unknown>;
export type RpcDebugTracingConfig = t.TypeOf<typeof rpcDebugTracingConfig>;
//# sourceMappingURL=debugTraceTransaction.d.ts.map