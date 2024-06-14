import { BytesLike } from "@ethersproject/bytes";
import { Coder, Reader, Result, Writer } from "./coders/abstract-coder";
import { ParamType } from "./fragments";
export declare type CoerceFunc = (type: string, value: any) => any;
export declare class AbiCoder {
    readonly coerceFunc: CoerceFunc;
    constructor(coerceFunc?: CoerceFunc);
    _getCoder(param: ParamType): Coder;
    _getWordSize(): number;
    _getReader(data: Uint8Array, allowLoose?: boolean): Reader;
    _getWriter(): Writer;
    getDefaultValue(types: ReadonlyArray<string | ParamType>): Result;
    encode(types: ReadonlyArray<string | ParamType>, values: ReadonlyArray<any>): string;
    decode(types: ReadonlyArray<string | ParamType>, data: BytesLike, loose?: boolean): Result;
}
export declare const defaultAbiCoder: AbiCoder;
//# sourceMappingURL=abi-coder.d.ts.map