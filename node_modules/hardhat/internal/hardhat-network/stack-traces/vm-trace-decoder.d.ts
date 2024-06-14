/// <reference types="node" />
import { TracingConfig } from "../provider/node-types";
import { ContractsIdentifier } from "./contracts-identifier";
import { MessageTrace } from "./message-trace";
import { Bytecode } from "./model";
export declare class VmTraceDecoder {
    private readonly _contractsIdentifier;
    constructor(_contractsIdentifier: ContractsIdentifier);
    getContractAndFunctionNamesForCall(code: Buffer, calldata?: Buffer): {
        contractName: string;
        functionName?: string;
    };
    tryToDecodeMessageTrace(messageTrace: MessageTrace): MessageTrace;
    addBytecode(bytecode: Bytecode): void;
}
export declare function initializeVmTraceDecoder(vmTraceDecoder: VmTraceDecoder, tracingConfig: TracingConfig): void;
//# sourceMappingURL=vm-trace-decoder.d.ts.map