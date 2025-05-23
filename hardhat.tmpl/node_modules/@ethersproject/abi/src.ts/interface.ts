"use strict";

import { getAddress } from "@ethersproject/address";
import { BigNumber, BigNumberish } from "@ethersproject/bignumber";
import { arrayify, BytesLike, concat, hexDataSlice, hexlify, hexZeroPad, isHexString } from "@ethersproject/bytes";
import { id } from "@ethersproject/hash";
import { keccak256 } from "@ethersproject/keccak256"
import { defineReadOnly, Description, getStatic } from "@ethersproject/properties";

import { AbiCoder, defaultAbiCoder } from "./abi-coder";
import { checkResultErrors, Result } from "./coders/abstract-coder";
import { ConstructorFragment, ErrorFragment, EventFragment, FormatTypes, Fragment, FunctionFragment, JsonFragment, ParamType } from "./fragments";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export { checkResultErrors, Result };

export class LogDescription extends Description<LogDescription> {
    readonly eventFragment: EventFragment;
    readonly name: string;
    readonly signature: string;
    readonly topic: string;
    readonly args: Result
}

export class TransactionDescription extends Description<TransactionDescription> {
    readonly functionFragment: FunctionFragment;
    readonly name: string;
    readonly args: Result;
    readonly signature: string;
    readonly sighash: string;
    readonly value: BigNumber;
}

export class ErrorDescription extends Description<ErrorDescription> {
    readonly errorFragment: ErrorFragment;
    readonly name: string;
    readonly args: Result;
    readonly signature: string;
    readonly sighash: string;
}

export class Indexed extends Description<Indexed> {
    readonly hash: string;
    readonly _isIndexed: boolean;

    static isIndexed(value: any): value is Indexed {
        return !!(value && value._isIndexed);
    }
}

const BuiltinErrors: Record<string, { signature: string, inputs: Array<string>, name: string, reason?: boolean }> = {
    "0x08c379a0": { signature: "Error(string)", name: "Error", inputs: [ "string" ], reason: true },
    "0x4e487b71": { signature: "Panic(uint256)", name: "Panic", inputs: [ "uint256" ] }
}

function wrapAccessError(property: string, error: Error): Error {
    const wrap = new Error(`deferred error during ABI decoding triggered accessing ${ property }`);
    (<any>wrap).error = error;
    return wrap;
}

/*
function checkNames(fragment: Fragment, type: "input" | "output", params: Array<ParamType>): void {
    params.reduce((accum, param) => {
        if (param.name) {
            if (accum[param.name]) {
                logger.throwArgumentError(`duplicate ${ type } parameter ${ JSON.stringify(param.name) } in ${ fragment.format("full") }`, "fragment", fragment);
            }
            accum[param.name] = true;
        }
        return accum;
    }, <{ [ name: string ]: boolean }>{ });
}
*/
export class Interface {
    readonly fragments: ReadonlyArray<Fragment>;

    readonly errors: { [ name: string ]: ErrorFragment };
    readonly events: { [ name: string ]: EventFragment };
    readonly functions: { [ name: string ]: FunctionFragment };
    readonly structs: { [ name: string ]: any };

    readonly deploy: ConstructorFragment;

    readonly _abiCoder: AbiCoder;

    readonly _isInterface: boolean;

    constructor(fragments: string | ReadonlyArray<Fragment | JsonFragment | string>) {
        let abi: ReadonlyArray<Fragment | JsonFragment | string> = [ ];
        if (typeof(fragments) === "string") {
            abi = JSON.parse(fragments);
        } else {
            abi = fragments;
        }

        defineReadOnly(this, "fragments", abi.map((fragment) => {
            return Fragment.from(fragment);
        }).filter((fragment) => (fragment != null)));

        defineReadOnly(this, "_abiCoder", getStatic<() => AbiCoder>(new.target, "getAbiCoder")());

        defineReadOnly(this, "functions", { });
        defineReadOnly(this, "errors", { });
        defineReadOnly(this, "events", { });
        defineReadOnly(this, "structs", { });

        // Add all fragments by their signature
        this.fragments.forEach((fragment) => {
            let bucket: { [ name: string ]: Fragment } = null;
            switch (fragment.type) {
                case "constructor":
                    if (this.deploy) {
                        logger.warn("duplicate definition - constructor");
                        return;
                    }
                    //checkNames(fragment, "input", fragment.inputs);
                    defineReadOnly(this, "deploy", <ConstructorFragment>fragment);
                    return;
                case "function":
                    //checkNames(fragment, "input", fragment.inputs);
                    //checkNames(fragment, "output", (<FunctionFragment>fragment).outputs);
                    bucket = this.functions;
                    break;
                case "event":
                    //checkNames(fragment, "input", fragment.inputs);
                    bucket = this.events;
                    break;
                case "error":
                    bucket = this.errors;
                    break;
                default:
                    return;
            }

            let signature = fragment.format();
            if (bucket[signature]) {
                logger.warn("duplicate definition - " + signature);
                return;
            }

            bucket[signature] = fragment;
        });

        // If we do not have a constructor add a default
        if (!this.deploy) {
            defineReadOnly(this, "deploy", ConstructorFragment.from({
                payable: false,
                type: "constructor"
            }));
        }

        defineReadOnly(this, "_isInterface", true);
    }

    format(format?: string): string | Array<string> {
        if (!format) { format = FormatTypes.full; }
        if (format === FormatTypes.sighash) {
            logger.throwArgumentError("interface does not support formatting sighash", "format", format);
        }

        const abi = this.fragments.map((fragment) => fragment.format(format));

        // We need to re-bundle the JSON fragments a bit
        if (format === FormatTypes.json) {
             return JSON.stringify(abi.map((j) => JSON.parse(j)));
        }

        return abi;
    }

    // Sub-classes can override these to handle other blockchains
    static getAbiCoder(): AbiCoder {
        return defaultAbiCoder;
    }

    static getAddress(address: string): string {
        return getAddress(address);
    }

    static getSighash(fragment: ErrorFragment | FunctionFragment): string {
        return hexDataSlice(id(fragment.format()), 0, 4);
    }

    static getEventTopic(eventFragment: EventFragment): string {
        return id(eventFragment.format());
    }

    // Find a function definition by any means necessary (unless it is ambiguous)
    getFunction(nameOrSignatureOrSighash: string): FunctionFragment {
        if (isHexString(nameOrSignatureOrSighash)) {
            for (const name in this.functions) {
                if (nameOrSignatureOrSighash === this.getSighash(name)) {
                    return this.functions[name];
                }
            }
            logger.throwArgumentError("no matching function", "sighash", nameOrSignatureOrSighash);
        }

        // It is a bare name, look up the function (will return null if ambiguous)
        if (nameOrSignatureOrSighash.indexOf("(") === -1) {
            const name = nameOrSignatureOrSighash.trim();
            const matching = Object.keys(this.functions).filter((f) => (f.split("("/* fix:) */)[0] === name));
            if (matching.length === 0) {
                logger.throwArgumentError("no matching function", "name", name);
            } else if (matching.length > 1) {
                logger.throwArgumentError("multiple matching functions", "name", name);
            }

            return this.functions[matching[0]];
        }

        // Normalize the signature and lookup the function
        const result = this.functions[FunctionFragment.fromString(nameOrSignatureOrSighash).format()];
        if (!result) {
            logger.throwArgumentError("no matching function", "signature", nameOrSignatureOrSighash);
        }
        return result;
    }

    // Find an event definition by any means necessary (unless it is ambiguous)
    getEvent(nameOrSignatureOrTopic: string): EventFragment {
        if (isHexString(nameOrSignatureOrTopic)) {
            const topichash = nameOrSignatureOrTopic.toLowerCase();
            for (const name in this.events) {
                if (topichash === this.getEventTopic(name)) {
                    return this.events[name];
                }
            }
            logger.throwArgumentError("no matching event", "topichash", topichash);
        }

        // It is a bare name, look up the function (will return null if ambiguous)
        if (nameOrSignatureOrTopic.indexOf("(") === -1) {
            const name = nameOrSignatureOrTopic.trim();
            const matching = Object.keys(this.events).filter((f) => (f.split("("/* fix:) */)[0] === name));
            if (matching.length === 0) {
                logger.throwArgumentError("no matching event", "name", name);
            } else if (matching.length > 1) {
                logger.throwArgumentError("multiple matching events", "name", name);
            }

            return this.events[matching[0]];
        }

        // Normalize the signature and lookup the function
        const result = this.events[EventFragment.fromString(nameOrSignatureOrTopic).format()];
        if (!result) {
            logger.throwArgumentError("no matching event", "signature", nameOrSignatureOrTopic);
        }
        return result;
    }

    // Find a function definition by any means necessary (unless it is ambiguous)
    getError(nameOrSignatureOrSighash: string): ErrorFragment {
        if (isHexString(nameOrSignatureOrSighash)) {
            const getSighash = getStatic<(f: ErrorFragment | FunctionFragment) => string>(this.constructor, "getSighash");
            for (const name in this.errors) {
                const error = this.errors[name];
                if (nameOrSignatureOrSighash === getSighash(error)) {
                    return this.errors[name];
                }
            }
            logger.throwArgumentError("no matching error", "sighash", nameOrSignatureOrSighash);
        }

        // It is a bare name, look up the function (will return null if ambiguous)
        if (nameOrSignatureOrSighash.indexOf("(") === -1) {
            const name = nameOrSignatureOrSighash.trim();
            const matching = Object.keys(this.errors).filter((f) => (f.split("("/* fix:) */)[0] === name));
            if (matching.length === 0) {
                logger.throwArgumentError("no matching error", "name", name);
            } else if (matching.length > 1) {
                logger.throwArgumentError("multiple matching errors", "name", name);
            }

            return this.errors[matching[0]];
        }

        // Normalize the signature and lookup the function
        const result = this.errors[FunctionFragment.fromString(nameOrSignatureOrSighash).format()];
        if (!result) {
            logger.throwArgumentError("no matching error", "signature", nameOrSignatureOrSighash);
        }
        return result;
    }

    // Get the sighash (the bytes4 selector) used by Solidity to identify a function
    getSighash(fragment: ErrorFragment | FunctionFragment | string): string {
        if (typeof(fragment) === "string") {
            try {
                fragment = this.getFunction(fragment);
            } catch (error) {
                try {
                    fragment = this.getError(<string>fragment);
                } catch (_) {
                    throw error;
                }
            }
        }

        return getStatic<(f: ErrorFragment | FunctionFragment) => string>(this.constructor, "getSighash")(fragment);
    }

    // Get the topic (the bytes32 hash) used by Solidity to identify an event
    getEventTopic(eventFragment: EventFragment | string): string {
        if (typeof(eventFragment) === "string") {
            eventFragment = this.getEvent(eventFragment);
        }

        return getStatic<(e: EventFragment) => string>(this.constructor, "getEventTopic")(eventFragment);
    }


    _decodeParams(params: ReadonlyArray<ParamType>, data: BytesLike): Result {
        return this._abiCoder.decode(params, data)
    }

    _encodeParams(params: ReadonlyArray<ParamType>, values: ReadonlyArray<any>): string {
        return this._abiCoder.encode(params, values)
    }

    encodeDeploy(values?: ReadonlyArray<any>): string {
        return this._encodeParams(this.deploy.inputs, values || [ ]);
    }

    decodeErrorResult(fragment: ErrorFragment | string, data: BytesLike): Result {
        if (typeof(fragment) === "string") {
            fragment = this.getError(fragment);
        }

        const bytes = arrayify(data);

        if (hexlify(bytes.slice(0, 4)) !== this.getSighash(fragment)) {
            logger.throwArgumentError(`data signature does not match error ${ fragment.name }.`, "data", hexlify(bytes));
        }

        return this._decodeParams(fragment.inputs, bytes.slice(4));
    }

    encodeErrorResult(fragment: ErrorFragment | string, values?: ReadonlyArray<any>): string {
        if (typeof(fragment) === "string") {
            fragment = this.getError(fragment);
        }

        return hexlify(concat([
            this.getSighash(fragment),
            this._encodeParams(fragment.inputs, values || [ ])
        ]));
    }

    // Decode the data for a function call (e.g. tx.data)
    decodeFunctionData(functionFragment: FunctionFragment | string, data: BytesLike): Result {
        if (typeof(functionFragment) === "string") {
            functionFragment = this.getFunction(functionFragment);
        }

        const bytes = arrayify(data);

        if (hexlify(bytes.slice(0, 4)) !== this.getSighash(functionFragment)) {
            logger.throwArgumentError(`data signature does not match function ${ functionFragment.name }.`, "data", hexlify(bytes));
        }

        return this._decodeParams(functionFragment.inputs, bytes.slice(4));
    }

    // Encode the data for a function call (e.g. tx.data)
    encodeFunctionData(functionFragment: FunctionFragment | string, values?: ReadonlyArray<any>): string {
        if (typeof(functionFragment) === "string") {
            functionFragment = this.getFunction(functionFragment);
        }

        return hexlify(concat([
            this.getSighash(functionFragment),
            this._encodeParams(functionFragment.inputs, values || [ ])
        ]));
    }

    // Decode the result from a function call (e.g. from eth_call)
    decodeFunctionResult(functionFragment: FunctionFragment | string, data: BytesLike): Result {
        if (typeof(functionFragment) === "string") {
            functionFragment = this.getFunction(functionFragment);
        }

        let bytes = arrayify(data);

        let reason: string = null;
        let message = "";
        let errorArgs: Result = null;
        let errorName: string = null;
        let errorSignature: string = null;
        switch (bytes.length % this._abiCoder._getWordSize()) {
            case 0:
                try {
                    return this._abiCoder.decode(functionFragment.outputs, bytes);
                } catch (error) { }
                break;

            case 4: {
                const selector = hexlify(bytes.slice(0, 4));
                const builtin = BuiltinErrors[selector];
                if (builtin) {
                    errorArgs = this._abiCoder.decode(builtin.inputs, bytes.slice(4));
                    errorName = builtin.name;
                    errorSignature = builtin.signature;
                    if (builtin.reason) { reason = errorArgs[0]; }
                    if (errorName === "Error") {
                        message = `; VM Exception while processing transaction: reverted with reason string ${ JSON.stringify(errorArgs[0]) }`;
                    } else if (errorName === "Panic") {
                        message = `; VM Exception while processing transaction: reverted with panic code ${ errorArgs[0] }`;
                    }
                } else {
                    try {
                        const error = this.getError(selector);
                        errorArgs = this._abiCoder.decode(error.inputs, bytes.slice(4));
                        errorName = error.name;
                        errorSignature = error.format();
                    } catch (error) { }
                }
                break;
            }
        }

        return logger.throwError("call revert exception" + message, Logger.errors.CALL_EXCEPTION, {
            method: functionFragment.format(),
            data: hexlify(data), errorArgs, errorName, errorSignature, reason
        });
    }

    // Encode the result for a function call (e.g. for eth_call)
    encodeFunctionResult(functionFragment: FunctionFragment | string, values?: ReadonlyArray<any>): string {
        if (typeof(functionFragment) === "string") {
            functionFragment = this.getFunction(functionFragment);
        }

        return hexlify(this._abiCoder.encode(functionFragment.outputs, values || [ ]));
    }

    // Create the filter for the event with search criteria (e.g. for eth_filterLog)
    encodeFilterTopics(eventFragment: EventFragment | string, values: ReadonlyArray<any>): Array<string | Array<string>> {
        if (typeof(eventFragment) === "string") {
            eventFragment = this.getEvent(eventFragment);
        }

        if (values.length > eventFragment.inputs.length) {
            logger.throwError("too many arguments for " + eventFragment.format(), Logger.errors.UNEXPECTED_ARGUMENT, {
                argument: "values",
                value: values
            })
        }

        let topics: Array<string | Array<string>> = [];
        if (!eventFragment.anonymous) { topics.push(this.getEventTopic(eventFragment)); }

        const encodeTopic = (param: ParamType, value: any): string => {
            if (param.type === "string") {
                 return id(value);
            } else if (param.type === "bytes") {
                 return keccak256(hexlify(value));
            }

            if (param.type === "bool" && typeof(value) === "boolean") {
                value = (value ? "0x01": "0x00");
            }

            if (param.type.match(/^u?int/)) {
                value = BigNumber.from(value).toHexString();
            }

            // Check addresses are valid
            if (param.type === "address") { this._abiCoder.encode( [ "address" ], [ value ]); }
            return hexZeroPad(hexlify(value), 32);
        };

        values.forEach((value, index) => {

            let param = (<EventFragment>eventFragment).inputs[index];

            if (!param.indexed) {
                if (value != null) {
                    logger.throwArgumentError("cannot filter non-indexed parameters; must be null", ("contract." + param.name), value);
                }
                return;
            }

            if (value == null) {
                topics.push(null);
            } else if (param.baseType === "array" || param.baseType === "tuple") {
                logger.throwArgumentError("filtering with tuples or arrays not supported", ("contract." + param.name), value);
            } else if (Array.isArray(value)) {
                topics.push(value.map((value) => encodeTopic(param, value)));
            } else {
                topics.push(encodeTopic(param, value));
            }
        });

        // Trim off trailing nulls
        while (topics.length && topics[topics.length - 1] === null) {
            topics.pop();
        }

        return topics;
    }

    encodeEventLog(eventFragment: EventFragment | string, values: ReadonlyArray<any>): { data: string, topics: Array<string> } {
        if (typeof(eventFragment) === "string") {
            eventFragment = this.getEvent(eventFragment);
        }

        const topics: Array<string> = [ ];

        const dataTypes: Array<ParamType> = [ ];
        const dataValues: Array<string> = [ ];

        if (!eventFragment.anonymous) {
            topics.push(this.getEventTopic(eventFragment));
        }

        if (values.length !== eventFragment.inputs.length) {
            logger.throwArgumentError("event arguments/values mismatch", "values", values);
        }

        eventFragment.inputs.forEach((param, index) => {
            const value = values[index];
            if (param.indexed) {
                if (param.type === "string") {
                    topics.push(id(value))
                } else if (param.type === "bytes") {
                    topics.push(keccak256(value))
                } else if (param.baseType === "tuple" || param.baseType === "array") {
                    // @TODO
                    throw new Error("not implemented");
                } else {
                    topics.push(this._abiCoder.encode([ param.type] , [ value ]));
                }
            } else {
                dataTypes.push(param);
                dataValues.push(value);
            }
        });

        return {
            data: this._abiCoder.encode(dataTypes , dataValues),
            topics: topics
        };
    }

    // Decode a filter for the event and the search criteria
    decodeEventLog(eventFragment: EventFragment | string, data: BytesLike, topics?: ReadonlyArray<string>): Result {
        if (typeof(eventFragment) === "string") {
            eventFragment = this.getEvent(eventFragment);
        }

        if (topics != null && !eventFragment.anonymous) {
            let topicHash = this.getEventTopic(eventFragment);
            if (!isHexString(topics[0], 32) || topics[0].toLowerCase() !== topicHash) {
                logger.throwError("fragment/topic mismatch", Logger.errors.INVALID_ARGUMENT, { argument: "topics[0]", expected: topicHash, value: topics[0] });
            }
            topics = topics.slice(1);
        }

        let indexed: Array<ParamType> = [];
        let nonIndexed: Array<ParamType> = [];
        let dynamic: Array<boolean> = [];

        eventFragment.inputs.forEach((param, index) => {
            if (param.indexed) {
                if (param.type === "string" || param.type === "bytes" || param.baseType === "tuple" || param.baseType === "array") {
                    indexed.push(ParamType.fromObject({ type: "bytes32", name: param.name }));
                    dynamic.push(true);
                } else {
                    indexed.push(param);
                    dynamic.push(false);
                }
            } else {
                nonIndexed.push(param);
                dynamic.push(false);
            }
        });

        let resultIndexed = (topics != null) ? this._abiCoder.decode(indexed, concat(topics)): null;
        let resultNonIndexed = this._abiCoder.decode(nonIndexed, data, true);

        let result: (Array<any> & { [ key: string ]: any }) = [ ];
        let nonIndexedIndex = 0, indexedIndex = 0;
        eventFragment.inputs.forEach((param, index) => {
            if (param.indexed) {
                if (resultIndexed == null) {
                    result[index] = new Indexed({ _isIndexed: true, hash: null });

                } else if (dynamic[index]) {
                    result[index] = new Indexed({ _isIndexed: true, hash: resultIndexed[indexedIndex++] });

                } else {
                    try {
                        result[index] = resultIndexed[indexedIndex++];
                    } catch (error) {
                        result[index] = error;
                    }
                }
            } else {
                try {
                    result[index] = resultNonIndexed[nonIndexedIndex++];
                } catch (error) {
                    result[index] = error;
                }
            }

            // Add the keyword argument if named and safe
            if (param.name && result[param.name] == null) {
                const value = result[index];

                // Make error named values throw on access
                if (value instanceof Error) {
                    Object.defineProperty(result, param.name, {
                        enumerable: true,
                        get: () => { throw wrapAccessError(`property ${ JSON.stringify(param.name) }`, value); }
                    });
                } else {
                    result[param.name] = value;
                }
            }
        });

        // Make all error indexed values throw on access
        for (let i = 0; i < result.length; i++) {
            const value = result[i];
            if (value instanceof Error) {
                Object.defineProperty(result, i, {
                    enumerable: true,
                    get: () => { throw wrapAccessError(`index ${ i }`, value); }
                });
            }
        }

        return Object.freeze(result);
    }

    // Given a transaction, find the matching function fragment (if any) and
    // determine all its properties and call parameters
    parseTransaction(tx: { data: string, value?: BigNumberish }): TransactionDescription {
        let fragment = this.getFunction(tx.data.substring(0, 10).toLowerCase())

        if (!fragment) { return null; }

        return new TransactionDescription({
            args: this._abiCoder.decode(fragment.inputs, "0x" + tx.data.substring(10)),
            functionFragment: fragment,
            name: fragment.name,
            signature: fragment.format(),
            sighash: this.getSighash(fragment),
            value: BigNumber.from(tx.value || "0"),
        });
    }

    // @TODO
    //parseCallResult(data: BytesLike): ??

    // Given an event log, find the matching event fragment (if any) and
    // determine all its properties and values
    parseLog(log: { topics: Array<string>, data: string}): LogDescription {
        let fragment = this.getEvent(log.topics[0]);

        if (!fragment || fragment.anonymous) { return null; }

        // @TODO: If anonymous, and the only method, and the input count matches, should we parse?
        //        Probably not, because just because it is the only event in the ABI does
        //        not mean we have the full ABI; maybe just a fragment?


       return new LogDescription({
            eventFragment: fragment,
            name: fragment.name,
            signature: fragment.format(),
            topic: this.getEventTopic(fragment),
            args: this.decodeEventLog(fragment, log.data, log.topics)
        });
    }

    parseError(data: BytesLike): ErrorDescription {
        const hexData = hexlify(data);
        let fragment = this.getError(hexData.substring(0, 10).toLowerCase())

        if (!fragment) { return null; }

        return new ErrorDescription({
            args: this._abiCoder.decode(fragment.inputs, "0x" + hexData.substring(10)),
            errorFragment: fragment,
            name: fragment.name,
            signature: fragment.format(),
            sighash: this.getSighash(fragment),
        });
    }


    /*
    static from(value: Array<Fragment | string | JsonAbi> | string | Interface) {
        if (Interface.isInterface(value)) {
            return value;
        }
        if (typeof(value) === "string") {
            return new Interface(JSON.parse(value));
        }
        return new Interface(value);
    }
    */

    static isInterface(value: any): value is Interface {
        return !!(value && value._isInterface);
    }
}

