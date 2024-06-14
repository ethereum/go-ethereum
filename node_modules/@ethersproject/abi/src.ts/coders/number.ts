"use strict";

import { BigNumber, BigNumberish } from "@ethersproject/bignumber";
import { MaxUint256, NegativeOne, One, Zero } from "@ethersproject/constants";

import { Coder, Reader, Writer } from "./abstract-coder";

export class NumberCoder extends Coder {
    readonly size: number;
    readonly signed: boolean;

    constructor(size: number, signed: boolean, localName: string) {
        const name = ((signed ? "int": "uint") + (size * 8));
        super(name, name, localName, false);

        this.size = size;
        this.signed = signed;
    }

    defaultValue(): number {
        return 0;
    }

    encode(writer: Writer, value: BigNumberish): number {
        let v = BigNumber.from(value);

        // Check bounds are safe for encoding
        let maxUintValue = MaxUint256.mask(writer.wordSize * 8);
        if (this.signed) {
            let bounds = maxUintValue.mask(this.size * 8 - 1);
            if (v.gt(bounds) || v.lt(bounds.add(One).mul(NegativeOne))) {
                this._throwError("value out-of-bounds", value);
            }
        } else if (v.lt(Zero) || v.gt(maxUintValue.mask(this.size * 8))) {
            this._throwError("value out-of-bounds", value);
        }

        v = v.toTwos(this.size * 8).mask(this.size * 8);

        if (this.signed) {
            v = v.fromTwos(this.size * 8).toTwos(8 * writer.wordSize);
        }

        return writer.writeValue(v);
    }

    decode(reader: Reader): any {
        let value = reader.readValue().mask(this.size * 8);

        if (this.signed) {
            value = value.fromTwos(this.size * 8);
        }

        return reader.coerce(this.name, value);
    }
}

