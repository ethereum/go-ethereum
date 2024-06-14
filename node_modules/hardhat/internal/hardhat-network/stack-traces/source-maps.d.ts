/// <reference types="node" />
import { Instruction, JumpType, SourceFile } from "./model";
export interface SourceMapLocation {
    offset: number;
    length: number;
    file: number;
}
export interface SourceMap {
    location: SourceMapLocation;
    jumpType: JumpType;
}
export declare function decodeInstructions(bytecode: Buffer, compressedSourcemaps: string, fileIdToSourceFile: Map<number, SourceFile>, isDeployment: boolean): Instruction[];
//# sourceMappingURL=source-maps.d.ts.map