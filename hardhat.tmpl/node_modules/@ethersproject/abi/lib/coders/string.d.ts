import { Reader, Writer } from "./abstract-coder";
import { DynamicBytesCoder } from "./bytes";
export declare class StringCoder extends DynamicBytesCoder {
    constructor(localName: string);
    defaultValue(): string;
    encode(writer: Writer, value: any): number;
    decode(reader: Reader): any;
}
//# sourceMappingURL=string.d.ts.map