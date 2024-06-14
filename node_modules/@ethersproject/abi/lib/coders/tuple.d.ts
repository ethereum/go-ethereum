import { Coder, Reader, Writer } from "./abstract-coder";
export declare class TupleCoder extends Coder {
    readonly coders: Array<Coder>;
    constructor(coders: Array<Coder>, localName: string);
    defaultValue(): any;
    encode(writer: Writer, value: Array<any> | {
        [name: string]: any;
    }): number;
    decode(reader: Reader): any;
}
//# sourceMappingURL=tuple.d.ts.map