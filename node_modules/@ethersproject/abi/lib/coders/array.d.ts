import { Coder, Reader, Result, Writer } from "./abstract-coder";
export declare function pack(writer: Writer, coders: ReadonlyArray<Coder>, values: Array<any> | {
    [name: string]: any;
}): number;
export declare function unpack(reader: Reader, coders: Array<Coder>): Result;
export declare class ArrayCoder extends Coder {
    readonly coder: Coder;
    readonly length: number;
    constructor(coder: Coder, length: number, localName: string);
    defaultValue(): Array<any>;
    encode(writer: Writer, value: Array<any>): number;
    decode(reader: Reader): any;
}
//# sourceMappingURL=array.d.ts.map