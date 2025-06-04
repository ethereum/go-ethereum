"use strict";

import { Coder, Reader, Writer } from "./abstract-coder";

export class NullCoder extends Coder {

    constructor(localName: string) {
        super("null", "", localName, false);
    }

    defaultValue(): null {
        return null;
    }

    encode(writer: Writer, value: any): number {
        if (value != null) { this._throwError("not null", value); }
        return writer.writeBytes([ ]);
    }

    decode(reader: Reader): any {
        reader.readBytes(0);
        return reader.coerce(this.name, null);
    }
}
