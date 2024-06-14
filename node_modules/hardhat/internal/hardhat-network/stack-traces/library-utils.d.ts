/// <reference types="node" />
import { CompilerOutputBytecode } from "../../../types";
export declare function getLibraryAddressPositions(bytecodeOutput: CompilerOutputBytecode): number[];
export declare function normalizeCompilerOutputBytecode(compilerOutputBytecodeObject: string, addressesPositions: number[]): Buffer;
export declare function linkHexStringBytecode(code: string, address: string, position: number): string;
export declare function zeroOutAddresses(code: Uint8Array, addressesPositions: number[]): Uint8Array;
export declare function zeroOutSlices(code: Uint8Array, slices: Array<{
    start: number;
    length: number;
}>): Uint8Array;
export declare function normalizeLibraryRuntimeBytecodeIfNecessary(code: Uint8Array): Uint8Array;
//# sourceMappingURL=library-utils.d.ts.map