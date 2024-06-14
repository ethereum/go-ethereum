"use strict";

// See: https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI

import { arrayify, BytesLike } from "@ethersproject/bytes";
import { defineReadOnly } from "@ethersproject/properties";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { Coder, Reader, Result, Writer } from "./coders/abstract-coder";
import { AddressCoder } from "./coders/address";
import { ArrayCoder } from "./coders/array";
import { BooleanCoder } from "./coders/boolean";
import { BytesCoder } from "./coders/bytes";
import { FixedBytesCoder } from "./coders/fixed-bytes";
import { NullCoder } from "./coders/null";
import { NumberCoder } from "./coders/number";
import { StringCoder } from "./coders/string";
import { TupleCoder } from "./coders/tuple";

import { ParamType } from "./fragments";


const paramTypeBytes = new RegExp(/^bytes([0-9]*)$/);
const paramTypeNumber = new RegExp(/^(u?int)([0-9]*)$/);


export type CoerceFunc = (type: string, value: any) => any;

export class AbiCoder {
    readonly coerceFunc: CoerceFunc;

    constructor(coerceFunc?: CoerceFunc) {
        defineReadOnly(this, "coerceFunc", coerceFunc || null);
    }

    _getCoder(param: ParamType): Coder {

        switch (param.baseType) {
            case "address":
                return new AddressCoder(param.name);
            case "bool":
                return new BooleanCoder(param.name);
            case "string":
                return new StringCoder(param.name);
            case "bytes":
                return new BytesCoder(param.name);
            case "array":
                return new ArrayCoder(this._getCoder(param.arrayChildren), param.arrayLength, param.name);
            case "tuple":
                return new TupleCoder((param.components || []).map((component) => {
                    return this._getCoder(component);
                }), param.name);
            case "":
                return new NullCoder(param.name);
        }

        // u?int[0-9]*
        let match = param.type.match(paramTypeNumber);
        if (match) {
            let size = parseInt(match[2] || "256");
            if (size === 0 || size > 256 || (size % 8) !== 0) {
                logger.throwArgumentError("invalid " + match[1] + " bit length", "param", param);
            }
            return new NumberCoder(size / 8, (match[1] === "int"), param.name);
        }

        // bytes[0-9]+
        match = param.type.match(paramTypeBytes);
        if (match) {
            let size = parseInt(match[1]);
            if (size === 0 || size > 32) {
                logger.throwArgumentError("invalid bytes length", "param", param);
            }
            return new FixedBytesCoder(size, param.name);
        }

        return logger.throwArgumentError("invalid type", "type", param.type);
    }

    _getWordSize(): number { return 32; }

    _getReader(data: Uint8Array, allowLoose?: boolean): Reader {
        return new Reader(data, this._getWordSize(), this.coerceFunc, allowLoose);
    }

    _getWriter(): Writer {
        return new Writer(this._getWordSize());
    }

    getDefaultValue(types: ReadonlyArray<string | ParamType>): Result {
        const coders: Array<Coder> = types.map((type) => this._getCoder(ParamType.from(type)));
        const coder = new TupleCoder(coders, "_");
        return coder.defaultValue();
    }

    encode(types: ReadonlyArray<string | ParamType>, values: ReadonlyArray<any>): string {
        if (types.length !== values.length) {
            logger.throwError("types/values length mismatch", Logger.errors.INVALID_ARGUMENT, {
                count: { types: types.length, values: values.length },
                value: { types: types, values: values }
            });
        }

        const coders = types.map((type) => this._getCoder(ParamType.from(type)));
        const coder = (new TupleCoder(coders, "_"));

        const writer = this._getWriter();
        coder.encode(writer, values);
        return writer.data;
    }

    decode(types: ReadonlyArray<string | ParamType>, data: BytesLike, loose?: boolean): Result {
        const coders: Array<Coder> = types.map((type) => this._getCoder(ParamType.from(type)));
        const coder = new TupleCoder(coders, "_");
        return coder.decode(this._getReader(arrayify(data), loose));
    }
}

export const defaultAbiCoder: AbiCoder = new AbiCoder();

