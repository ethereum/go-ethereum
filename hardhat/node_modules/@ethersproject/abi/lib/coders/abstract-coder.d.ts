import { BytesLike } from "@ethersproject/bytes";
import { BigNumber, BigNumberish } from "@ethersproject/bignumber";
export interface Result extends ReadonlyArray<any> {
    readonly [key: string]: any;
}
export declare function checkResultErrors(result: Result): Array<{
    path: Array<string | number>;
    error: Error;
}>;
export declare type CoerceFunc = (type: string, value: any) => any;
export declare abstract class Coder {
    readonly name: string;
    readonly type: string;
    readonly localName: string;
    readonly dynamic: boolean;
    constructor(name: string, type: string, localName: string, dynamic: boolean);
    _throwError(message: string, value: any): void;
    abstract encode(writer: Writer, value: any): number;
    abstract decode(reader: Reader): any;
    abstract defaultValue(): any;
}
export declare class Writer {
    readonly wordSize: number;
    _data: Array<Uint8Array>;
    _dataLength: number;
    _padding: Uint8Array;
    constructor(wordSize?: number);
    get data(): string;
    get length(): number;
    _writeData(data: Uint8Array): number;
    appendWriter(writer: Writer): number;
    writeBytes(value: BytesLike): number;
    _getValue(value: BigNumberish): Uint8Array;
    writeValue(value: BigNumberish): number;
    writeUpdatableValue(): (value: BigNumberish) => void;
}
export declare class Reader {
    readonly wordSize: number;
    readonly allowLoose: boolean;
    readonly _data: Uint8Array;
    readonly _coerceFunc: CoerceFunc;
    _offset: number;
    constructor(data: BytesLike, wordSize?: number, coerceFunc?: CoerceFunc, allowLoose?: boolean);
    get data(): string;
    get consumed(): number;
    static coerce(name: string, value: any): any;
    coerce(name: string, value: any): any;
    _peekBytes(offset: number, length: number, loose?: boolean): Uint8Array;
    subReader(offset: number): Reader;
    readBytes(length: number, loose?: boolean): Uint8Array;
    readValue(): BigNumber;
}
//# sourceMappingURL=abstract-coder.d.ts.map