"use strict";
import { toUtf8Bytes, toUtf8String } from "@ethersproject/strings";
import { DynamicBytesCoder } from "./bytes";
export class StringCoder extends DynamicBytesCoder {
    constructor(localName) {
        super("string", localName);
    }
    defaultValue() {
        return "";
    }
    encode(writer, value) {
        return super.encode(writer, toUtf8Bytes(value));
    }
    decode(reader) {
        return toUtf8String(super.decode(reader));
    }
}
//# sourceMappingURL=string.js.map