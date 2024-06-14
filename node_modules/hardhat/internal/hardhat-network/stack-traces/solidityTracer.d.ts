import { MessageTrace } from "./message-trace";
import { SolidityStackTrace } from "./solidity-stack-trace";
export declare class SolidityTracer {
    private _errorInferrer;
    getStackTrace(maybeDecodedMessageTrace: MessageTrace): SolidityStackTrace;
    private _getCallMessageStackTrace;
    private _getUnrecognizedMessageStackTrace;
    private _getCreateMessageStackTrace;
    private _getPrecompileMessageStackTrace;
    private _traceEvmExecution;
    private _rawTraceEvmExecution;
    private _getLastSubtrace;
}
//# sourceMappingURL=solidityTracer.d.ts.map