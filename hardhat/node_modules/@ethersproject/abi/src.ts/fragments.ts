"use strict";

import { BigNumber } from "@ethersproject/bignumber";
import { defineReadOnly } from "@ethersproject/properties";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

export interface JsonFragmentType {
    readonly name?: string;
    readonly indexed?: boolean;
    readonly type?: string;
    readonly internalType?: any; // @TODO: in v6 reduce type
    readonly components?: ReadonlyArray<JsonFragmentType>;
}

export interface JsonFragment {
    readonly name?: string;
    readonly type?: string;

    readonly anonymous?: boolean;

    readonly payable?: boolean;
    readonly constant?: boolean;
    readonly stateMutability?: string;

    readonly inputs?: ReadonlyArray<JsonFragmentType>;
    readonly outputs?: ReadonlyArray<JsonFragmentType>;

    readonly gas?: string;
};

const _constructorGuard = { };

// AST Node parser state
type ParseState = {
    allowArray?: boolean,
    allowName?: boolean,
    allowParams?: boolean,
    allowType?: boolean,
    readArray?: boolean,
};

// AST Node
type ParseNode = {
    parent?: any,
    type?: string,
    name?: string,
    state?: ParseState,
    indexed?: boolean,
    components?: Array<ParseNode>
};

let ModifiersBytes: { [ name: string ]: boolean } = { calldata: true, memory: true, storage: true };
let ModifiersNest: { [ name: string ]: boolean } = { calldata: true, memory: true };
function checkModifier(type: string, name: string): boolean {
    if (type === "bytes" || type === "string") {
        if (ModifiersBytes[name]) { return true; }
    } else if (type === "address") {
        if (name === "payable") { return true; }
    } else if (type.indexOf("[") >= 0 || type === "tuple") {
        if (ModifiersNest[name]) { return true; }
    }
    if (ModifiersBytes[name] || name === "payable") {
        logger.throwArgumentError("invalid modifier", "name", name);
    }
    return false;
}

// @TODO: Make sure that children of an indexed tuple are marked with a null indexed
function parseParamType(param: string, allowIndexed: boolean): ParseNode {

    let originalParam = param;
    function throwError(i: number) {
        logger.throwArgumentError(`unexpected character at position ${ i }`, "param", param);
    }
    param = param.replace(/\s/g, " ");

    function newNode(parent: ParseNode): ParseNode {
        let node: ParseNode = { type: "", name: "", parent: parent, state: { allowType: true } };
        if (allowIndexed) { node.indexed = false; }
        return node
    }

    let parent: ParseNode = { type: "", name: "", state: { allowType: true } };
    let node = parent;

    for (let i = 0; i < param.length; i++) {
        let c = param[i];
        switch (c) {
            case "(":
                if (node.state.allowType && node.type === "") {
                    node.type = "tuple";
                } else if (!node.state.allowParams) {
                    throwError(i);
                }
                node.state.allowType = false;
                node.type = verifyType(node.type);
                node.components = [ newNode(node) ];
                node = node.components[0];
                break;

            case ")":
                delete node.state;

                if (node.name === "indexed") {
                    if (!allowIndexed) { throwError(i); }
                    node.indexed = true;
                    node.name = "";
                }

                if (checkModifier(node.type, node.name)) { node.name = ""; }

                node.type = verifyType(node.type);

                let child = node;
                node = node.parent;
                if (!node) { throwError(i); }
                delete child.parent;
                node.state.allowParams = false;
                node.state.allowName = true;
                node.state.allowArray = true;
                break;

            case ",":
                delete node.state;

                if (node.name === "indexed") {
                    if (!allowIndexed) { throwError(i); }
                    node.indexed = true;
                    node.name = "";
                }

                if (checkModifier(node.type, node.name)) { node.name = ""; }

                node.type = verifyType(node.type);

                let sibling: ParseNode = newNode(node.parent);
                 //{ type: "", name: "", parent: node.parent, state: { allowType: true } };
                node.parent.components.push(sibling);
                delete node.parent;
                node = sibling;
                break;

            // Hit a space...
            case " ":

                // If reading type, the type is done and may read a param or name
                if (node.state.allowType) {
                    if (node.type !== "") {
                        node.type = verifyType(node.type);
                        delete node.state.allowType;
                        node.state.allowName = true;
                        node.state.allowParams = true;
                    }
                }

                // If reading name, the name is done
                if (node.state.allowName) {
                    if (node.name !== "") {
                        if (node.name === "indexed") {
                            if (!allowIndexed) { throwError(i); }
                            if (node.indexed) { throwError(i); }
                            node.indexed = true;
                            node.name = "";
                        } else if (checkModifier(node.type, node.name)) {
                            node.name = "";
                        } else {
                            node.state.allowName = false;
                        }
                    }
                }

                break;

            case "[":
                if (!node.state.allowArray) { throwError(i); }

                node.type += c;

                node.state.allowArray = false;
                node.state.allowName = false;
                node.state.readArray = true;
                break;

            case "]":
                if (!node.state.readArray) { throwError(i); }

                node.type += c;

                node.state.readArray = false;
                node.state.allowArray = true;
                node.state.allowName = true;
                break;

            default:
                if (node.state.allowType) {
                    node.type += c;
                    node.state.allowParams = true;
                    node.state.allowArray = true;
                } else if (node.state.allowName) {
                    node.name += c;
                    delete node.state.allowArray;
                } else if (node.state.readArray) {
                    node.type += c;
                } else {
                    throwError(i);
               }
        }
    }

    if (node.parent) { logger.throwArgumentError("unexpected eof", "param", param); }

    delete parent.state;

    if (node.name === "indexed") {
        if (!allowIndexed) { throwError(originalParam.length - 7); }
        if (node.indexed) { throwError(originalParam.length - 7); }
        node.indexed = true;
        node.name = "";
    } else if (checkModifier(node.type, node.name)) {
        node.name = "";
    }

    parent.type = verifyType(parent.type);

    return parent;
}

function populate(object: any, params: any) {
    for (let key in params) { defineReadOnly(object, key, params[key]); }
}

export const FormatTypes: { [ name: string ]: string } = Object.freeze({
    // Bare formatting, as is needed for computing a sighash of an event or function
    sighash: "sighash",

    // Human-Readable with Minimal spacing and without names (compact human-readable)
    minimal: "minimal",

    // Human-Readable with nice spacing, including all names
    full: "full",

    // JSON-format a la Solidity
    json: "json"
});

const paramTypeArray = new RegExp(/^(.*)\[([0-9]*)\]$/);

export class ParamType {

    // The local name of the parameter (of null if unbound)
    readonly name: string;

    // The fully qualified type (e.g. "address", "tuple(address)", "uint256[3][]"
    readonly type: string;

    // The base type (e.g. "address", "tuple", "array")
    readonly baseType: string;

    // Indexable Paramters ONLY (otherwise null)
    readonly indexed: boolean;

    // Tuples ONLY: (otherwise null)
    //  - sub-components
    readonly components: Array<ParamType>;

    // Arrays ONLY: (otherwise null)
    //  - length of the array (-1 for dynamic length)
    //  - child type
    readonly arrayLength: number;
    readonly arrayChildren: ParamType;

    readonly _isParamType: boolean;

    constructor(constructorGuard: any, params: any) {
        if (constructorGuard !== _constructorGuard) { logger.throwError("use fromString", Logger.errors.UNSUPPORTED_OPERATION, {
            operation: "new ParamType()"
        }); }
        populate(this, params);

        let match = this.type.match(paramTypeArray);
        if (match) {
            populate(this, {
                arrayLength: parseInt(match[2] || "-1"),
                arrayChildren: ParamType.fromObject({
                    type: match[1],
                    components: this.components
                }),
                baseType: "array"
            });
        } else {
            populate(this, {
                arrayLength: null,
                arrayChildren: null,
                baseType: ((this.components != null) ? "tuple": this.type)
            });
        }

        this._isParamType = true;

        Object.freeze(this);
    }

    // Format the parameter fragment
    //   - sighash: "(uint256,address)"
    //   - minimal: "tuple(uint256,address) indexed"
    //   - full:    "tuple(uint256 foo, address bar) indexed baz"
    format(format?: string): string {
        if (!format) { format = FormatTypes.sighash; }
        if (!FormatTypes[format]) {
            logger.throwArgumentError("invalid format type", "format", format);
        }

        if (format === FormatTypes.json) {
            let result: any = {
                type: ((this.baseType === "tuple") ? "tuple": this.type),
                name: (this.name || undefined)
            };
            if (typeof(this.indexed) === "boolean") { result.indexed = this.indexed; }
            if (this.components) {
                result.components = this.components.map((comp) => JSON.parse(comp.format(format)));
            }
            return JSON.stringify(result);
        }

        let result = "";

        // Array
        if (this.baseType === "array") {
            result += this.arrayChildren.format(format);
            result += "[" + (this.arrayLength < 0 ? "": String(this.arrayLength)) + "]";
        } else {
            if (this.baseType === "tuple") {
                if (format !== FormatTypes.sighash) {
                    result += this.type;
                }
                result += "(" + this.components.map(
                    (comp) => comp.format(format)
                ).join((format === FormatTypes.full) ? ", ": ",") + ")";
            } else {
                result += this.type;
            }
        }

        if (format !== FormatTypes.sighash) {
            if (this.indexed === true) { result += " indexed"; }
            if (format === FormatTypes.full && this.name) {
                result += " " + this.name;
            }
        }

        return result;
    }

    static from(value: string | JsonFragmentType | ParamType, allowIndexed?: boolean): ParamType {
        if (typeof(value) === "string") {
            return ParamType.fromString(value, allowIndexed);
        }
        return ParamType.fromObject(value);
    }

    static fromObject(value: JsonFragmentType | ParamType): ParamType {
        if (ParamType.isParamType(value)) { return value; }

        return new ParamType(_constructorGuard, {
            name: (value.name || null),
            type: verifyType(value.type),
            indexed: ((value.indexed == null) ? null: !!value.indexed),
            components: (value.components ? value.components.map(ParamType.fromObject): null)
        });
    }

    static fromString(value: string, allowIndexed?: boolean): ParamType {
        function ParamTypify(node: ParseNode): ParamType {
            return ParamType.fromObject({
                name: node.name,
                type: node.type,
                indexed: node.indexed,
                components: node.components
            });
        }

        return ParamTypify(parseParamType(value, !!allowIndexed));
    }

    static isParamType(value: any): value is ParamType {
        return !!(value != null && value._isParamType);
    }
};

function parseParams(value: string, allowIndex: boolean): Array<ParamType> {
    return splitNesting(value).map((param) => ParamType.fromString(param, allowIndex));
}

type TypeCheck<T> = { -readonly [ K in keyof T ]: T[K] };

interface _Fragment {
    readonly type: string;
    readonly name: string;
    readonly inputs: ReadonlyArray<ParamType>;
}

export abstract class Fragment {

    readonly type: string;
    readonly name: string;
    readonly inputs: Array<ParamType>;

    readonly _isFragment: boolean;

    constructor(constructorGuard: any, params: any) {
        if (constructorGuard !== _constructorGuard) {
            logger.throwError("use a static from method", Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "new Fragment()"
            });
        }
        populate(this, params);

        this._isFragment = true;

        Object.freeze(this);
    }

    abstract format(format?: string): string;

    static from(value: Fragment | JsonFragment | string): Fragment {
        if (Fragment.isFragment(value)) { return value; }

        if (typeof(value) === "string") {
            return Fragment.fromString(value);
        }

        return Fragment.fromObject(value);
    }

    static fromObject(value: Fragment | JsonFragment): Fragment {
        if (Fragment.isFragment(value)) { return value; }

        switch (value.type) {
            case "function":
                return FunctionFragment.fromObject(value);
            case "event":
                return EventFragment.fromObject(value);
            case "constructor":
                return ConstructorFragment.fromObject(value);
            case "error":
                return ErrorFragment.fromObject(value);
            case "fallback":
            case "receive":
                // @TODO: Something? Maybe return a FunctionFragment? A custom DefaultFunctionFragment?
                return null;
        }

        return logger.throwArgumentError("invalid fragment object", "value", value);
    }

    static fromString(value: string): Fragment {
        // Make sure the "returns" is surrounded by a space and all whitespace is exactly one space
        value = value.replace(/\s/g, " ");
        value = value.replace(/\(/g, " (").replace(/\)/g, ") ").replace(/\s+/g, " ");
        value = value.trim();

        if (value.split(" ")[0] === "event") {
           return EventFragment.fromString(value.substring(5).trim());
        } else if (value.split(" ")[0] === "function") {
            return FunctionFragment.fromString(value.substring(8).trim());
        } else if (value.split("(")[0].trim() === "constructor") {
            return ConstructorFragment.fromString(value.trim());
        } else if (value.split(" ")[0] === "error") {
           return ErrorFragment.fromString(value.substring(5).trim());
        }

        return logger.throwArgumentError("unsupported fragment", "value", value);
    }

    static isFragment(value: any): value is Fragment {
        return !!(value && value._isFragment);
    }
}

interface _EventFragment extends _Fragment {
    readonly anonymous: boolean;
}

export class EventFragment extends Fragment {
    readonly anonymous: boolean;

    format(format?: string): string {
        if (!format) { format = FormatTypes.sighash; }
        if (!FormatTypes[format]) {
            logger.throwArgumentError("invalid format type", "format", format);
        }

        if (format === FormatTypes.json) {
            return JSON.stringify({
                type: "event",
                anonymous: this.anonymous,
                name: this.name,
                inputs: this.inputs.map((input) => JSON.parse(input.format(format)))
            });
        }

        let result = "";

        if (format !== FormatTypes.sighash) {
            result += "event ";
        }

        result += this.name + "(" + this.inputs.map(
            (input) => input.format(format)
        ).join((format === FormatTypes.full) ? ", ": ",") + ") ";

        if (format !== FormatTypes.sighash) {
            if (this.anonymous) {
                result += "anonymous ";
            }
        }

        return result.trim();
    }

    static from(value: EventFragment | JsonFragment | string): EventFragment {
        if (typeof(value) === "string") {
            return EventFragment.fromString(value);
        }
        return EventFragment.fromObject(value);
    }

    static fromObject(value: JsonFragment | EventFragment): EventFragment {
        if (EventFragment.isEventFragment(value)) { return value; }

        if (value.type !== "event") {
            logger.throwArgumentError("invalid event object", "value", value);
        }

        const params: TypeCheck<_EventFragment> = {
            name: verifyIdentifier(value.name),
            anonymous: value.anonymous,
            inputs: (value.inputs ? value.inputs.map(ParamType.fromObject) : []),
            type: "event"
        };

        return new EventFragment(_constructorGuard, params);
    }

    static fromString(value: string): EventFragment {

        let match = value.match(regexParen);
        if (!match) {
            logger.throwArgumentError("invalid event string", "value", value);
        }

        let anonymous = false;
        match[3].split(" ").forEach((modifier) => {
            switch(modifier.trim()) {
                case "anonymous":
                    anonymous = true;
                    break;
                case "":
                    break;
                default:
                    logger.warn("unknown modifier: " + modifier);
            }
        });

        return EventFragment.fromObject({
            name: match[1].trim(),
            anonymous: anonymous,
            inputs: parseParams(match[2], true),
            type: "event"
        });
    }

    static isEventFragment(value: any): value is EventFragment {
        return (value && value._isFragment && value.type === "event");
    }
}

function parseGas(value: string, params: any): string {
    params.gas = null;

    let comps = value.split("@");
    if (comps.length !== 1) {
        if (comps.length > 2) {
            logger.throwArgumentError("invalid human-readable ABI signature", "value", value);
        }
        if (!comps[1].match(/^[0-9]+$/)) {
            logger.throwArgumentError("invalid human-readable ABI signature gas", "value", value);
        }
        params.gas = BigNumber.from(comps[1]);
        return comps[0];
    }

    return value;
}

function parseModifiers(value: string, params: any): void {
    params.constant = false;
    params.payable = false;
    params.stateMutability = "nonpayable";

    value.split(" ").forEach((modifier) => {
        switch (modifier.trim()) {
            case "constant":
                params.constant = true;
                break;
            case "payable":
                params.payable = true;
                params.stateMutability = "payable";
                break;
            case "nonpayable":
                params.payable = false;
                params.stateMutability = "nonpayable";
                break;
            case "pure":
                params.constant = true;
                params.stateMutability = "pure";
                break;
            case "view":
                params.constant = true;
                params.stateMutability = "view";
                break;
            case "external":
            case "public":
            case "":
                break;
            default:
                console.log("unknown modifier: " + modifier);
        }
    });
}

type StateInputValue = {
    constant?: boolean;
    payable?: boolean;
    stateMutability?: string;
    type?: string;
};

type StateOutputValue = {
    constant: boolean;
    payable: boolean;
    stateMutability: string;
};

function verifyState(value: StateInputValue): StateOutputValue {
    let result: any = {
        constant: false,
        payable: true,
        stateMutability: "payable"
    };

    if (value.stateMutability != null) {
        result.stateMutability = value.stateMutability;

        // Set (and check things are consistent) the constant property
        result.constant = (result.stateMutability === "view" || result.stateMutability === "pure");
        if (value.constant != null) {
            if ((!!value.constant) !== result.constant) {
                logger.throwArgumentError("cannot have constant function with mutability " + result.stateMutability, "value", value);
            }
        }

        // Set (and check things are consistent) the payable property
        result.payable = (result.stateMutability === "payable");
        if (value.payable != null) {
            if ((!!value.payable) !== result.payable) {
                logger.throwArgumentError("cannot have payable function with mutability " + result.stateMutability, "value", value);
            }
        }

    } else if (value.payable != null) {
        result.payable = !!value.payable;

        // If payable we can assume non-constant; otherwise we can't assume
        if (value.constant == null && !result.payable && value.type !== "constructor") {
            logger.throwArgumentError("unable to determine stateMutability", "value", value);
        }

        result.constant = !!value.constant;

        if (result.constant) {
            result.stateMutability = "view";
        } else {
            result.stateMutability = (result.payable ? "payable": "nonpayable");
        }

        if (result.payable && result.constant) {
            logger.throwArgumentError("cannot have constant payable function", "value", value);
        }

    } else if (value.constant != null) {
        result.constant = !!value.constant;
        result.payable = !result.constant;
        result.stateMutability = (result.constant ? "view": "payable");

    } else if (value.type !== "constructor") {
        logger.throwArgumentError("unable to determine stateMutability", "value", value);
    }

    return result;
}

interface _ConstructorFragment extends _Fragment {
    stateMutability: string;
    payable: boolean;
    gas?: BigNumber;
}

export class ConstructorFragment extends Fragment {
    stateMutability: string;
    payable: boolean;
    gas?: BigNumber;

    format(format?: string): string {
        if (!format) { format = FormatTypes.sighash; }
        if (!FormatTypes[format]) {
            logger.throwArgumentError("invalid format type", "format", format);
        }

        if (format === FormatTypes.json) {
            return JSON.stringify({
                type: "constructor",
                stateMutability: ((this.stateMutability !== "nonpayable") ? this.stateMutability: undefined),
                payable: this.payable,
                gas: (this.gas ? this.gas.toNumber(): undefined),
                inputs: this.inputs.map((input) => JSON.parse(input.format(format)))
            });
        }

        if (format === FormatTypes.sighash) {
            logger.throwError("cannot format a constructor for sighash", Logger.errors.UNSUPPORTED_OPERATION, {
                operation: "format(sighash)"
            });
        }

        let result = "constructor(" + this.inputs.map(
            (input) => input.format(format)
        ).join((format === FormatTypes.full) ? ", ": ",") + ") ";

        if (this.stateMutability && this.stateMutability !== "nonpayable") {
            result += this.stateMutability + " ";
        }

        return result.trim();
    }

    static from(value: ConstructorFragment | JsonFragment | string): ConstructorFragment {
        if (typeof(value) === "string") {
            return ConstructorFragment.fromString(value);
        }
        return ConstructorFragment.fromObject(value);
    }

    static fromObject(value: ConstructorFragment | JsonFragment): ConstructorFragment {
        if (ConstructorFragment.isConstructorFragment(value)) { return value; }

        if (value.type !== "constructor") {
            logger.throwArgumentError("invalid constructor object", "value", value);
        }

        let state = verifyState(value);
        if (state.constant) {
            logger.throwArgumentError("constructor cannot be constant", "value", value);
        }

        const params: TypeCheck<_ConstructorFragment> = {
            name: null,
            type: value.type,
            inputs: (value.inputs ? value.inputs.map(ParamType.fromObject): []),
            payable: state.payable,
            stateMutability: state.stateMutability,
            gas: (value.gas ? BigNumber.from(value.gas): null)
        };

        return new ConstructorFragment(_constructorGuard, params);
    }

    static fromString(value: string): ConstructorFragment {
        let params: any = { type: "constructor" };

        value = parseGas(value, params);

        let parens = value.match(regexParen);
        if (!parens || parens[1].trim() !== "constructor") {
            logger.throwArgumentError("invalid constructor string", "value", value);
        }

        params.inputs = parseParams(parens[2].trim(), false);

        parseModifiers(parens[3].trim(), params);

        return ConstructorFragment.fromObject(params);
    }

    static isConstructorFragment(value: any): value is ConstructorFragment {
        return (value && value._isFragment && value.type === "constructor");
    }
}

interface _FunctionFragment extends _ConstructorFragment {
    constant: boolean;
    outputs?: Array<ParamType>;
}

export class FunctionFragment extends ConstructorFragment {
    constant: boolean;
    outputs?: Array<ParamType>;

    format(format?: string): string {
        if (!format) { format = FormatTypes.sighash; }
        if (!FormatTypes[format]) {
            logger.throwArgumentError("invalid format type", "format", format);
        }

        if (format === FormatTypes.json) {
            return JSON.stringify({
                type: "function",
                name: this.name,
                constant: this.constant,
                stateMutability: ((this.stateMutability !== "nonpayable") ? this.stateMutability: undefined),
                payable: this.payable,
                gas: (this.gas ? this.gas.toNumber(): undefined),
                inputs: this.inputs.map((input) => JSON.parse(input.format(format))),
                outputs: this.outputs.map((output) => JSON.parse(output.format(format))),
            });
        }

        let result = "";

        if (format !== FormatTypes.sighash) {
            result += "function ";
        }

        result += this.name + "(" + this.inputs.map(
            (input) => input.format(format)
        ).join((format === FormatTypes.full) ? ", ": ",") + ") ";

        if (format !== FormatTypes.sighash) {
            if (this.stateMutability) {
                if (this.stateMutability !== "nonpayable") {
                    result += (this.stateMutability + " ");
                }
            } else if (this.constant) {
                result += "view ";
            }

            if (this.outputs && this.outputs.length) {
                result += "returns (" + this.outputs.map(
                    (output) => output.format(format)
                ).join(", ") + ") ";
            }

            if (this.gas != null) {
                result += "@" + this.gas.toString() + " ";
            }
        }

        return result.trim();
    }

    static from(value: FunctionFragment | JsonFragment | string): FunctionFragment {
        if (typeof(value) === "string") {
            return FunctionFragment.fromString(value);
        }
        return FunctionFragment.fromObject(value);
    }

    static fromObject(value: FunctionFragment | JsonFragment): FunctionFragment {
        if (FunctionFragment.isFunctionFragment(value)) { return value; }

        if (value.type !== "function") {
            logger.throwArgumentError("invalid function object", "value", value);
        }

        let state = verifyState(value);

        const params: TypeCheck<_FunctionFragment> = {
            type: value.type,
            name: verifyIdentifier(value.name),
            constant: state.constant,
            inputs: (value.inputs ? value.inputs.map(ParamType.fromObject): []),
            outputs: (value.outputs ? value.outputs.map(ParamType.fromObject): [ ]),
            payable: state.payable,
            stateMutability: state.stateMutability,
            gas: (value.gas ? BigNumber.from(value.gas): null)
        };

        return new FunctionFragment(_constructorGuard, params);
    }

    static fromString(value: string): FunctionFragment {
        let params: any = { type: "function" };
        value = parseGas(value, params);

        let comps = value.split(" returns ");
        if (comps.length > 2) {
            logger.throwArgumentError("invalid function string", "value", value);
        }

        let parens = comps[0].match(regexParen);
        if (!parens) {
            logger.throwArgumentError("invalid function signature", "value", value);
        }

        params.name = parens[1].trim();
        if (params.name) { verifyIdentifier(params.name); }

        params.inputs = parseParams(parens[2], false);

        parseModifiers(parens[3].trim(), params);

        // We have outputs
        if (comps.length > 1) {
           let returns = comps[1].match(regexParen);
            if (returns[1].trim() != "" || returns[3].trim() != "") {
                logger.throwArgumentError("unexpected tokens", "value", value);
            }
            params.outputs = parseParams(returns[2], false);
        } else {
            params.outputs = [ ];
        }

        return FunctionFragment.fromObject(params);
    }

    static isFunctionFragment(value: any): value is FunctionFragment {
        return (value && value._isFragment && value.type === "function");
    }
}

//export class StructFragment extends Fragment {
//}

function checkForbidden(fragment: ErrorFragment): ErrorFragment {
    const sig = fragment.format();
    if (sig === "Error(string)" || sig === "Panic(uint256)") {
        logger.throwArgumentError(`cannot specify user defined ${ sig } error`, "fragment", fragment);
    }
    return fragment;
}

export class ErrorFragment extends Fragment {

    format(format?: string): string {
        if (!format) { format = FormatTypes.sighash; }
        if (!FormatTypes[format]) {
            logger.throwArgumentError("invalid format type", "format", format);
        }

        if (format === FormatTypes.json) {
            return JSON.stringify({
                type: "error",
                name: this.name,
                inputs: this.inputs.map((input) => JSON.parse(input.format(format))),
            });
        }

        let result = "";

        if (format !== FormatTypes.sighash) {
            result += "error ";
        }

        result += this.name + "(" + this.inputs.map(
            (input) => input.format(format)
        ).join((format === FormatTypes.full) ? ", ": ",") + ") ";

        return result.trim();
    }

    static from(value: ErrorFragment | JsonFragment | string): ErrorFragment {
        if (typeof(value) === "string") {
            return ErrorFragment.fromString(value);
        }
        return ErrorFragment.fromObject(value);
    }

    static fromObject(value: ErrorFragment | JsonFragment): ErrorFragment {
        if (ErrorFragment.isErrorFragment(value)) { return value; }

        if (value.type !== "error") {
            logger.throwArgumentError("invalid error object", "value", value);
        }

        const params: TypeCheck<_Fragment> = {
            type: value.type,
            name: verifyIdentifier(value.name),
            inputs: (value.inputs ? value.inputs.map(ParamType.fromObject): [])
        };

        return checkForbidden(new ErrorFragment(_constructorGuard, params));
    }

    static fromString(value: string): ErrorFragment {
        let params: any = { type: "error" };

        let parens = value.match(regexParen);
        if (!parens) {
            logger.throwArgumentError("invalid error signature", "value", value);
        }

        params.name = parens[1].trim();
        if (params.name) { verifyIdentifier(params.name); }

        params.inputs = parseParams(parens[2], false);

        return checkForbidden(ErrorFragment.fromObject(params));
    }

    static isErrorFragment(value: any): value is ErrorFragment {
        return (value && value._isFragment && value.type === "error");
    }
}

function verifyType(type: string): string {

    // These need to be transformed to their full description
    if (type.match(/^uint($|[^1-9])/)) {
        type = "uint256" + type.substring(4);
    } else if (type.match(/^int($|[^1-9])/)) {
        type = "int256" + type.substring(3);
    }

    // @TODO: more verification

    return type;
}

// See: https://github.com/ethereum/solidity/blob/1f8f1a3db93a548d0555e3e14cfc55a10e25b60e/docs/grammar/SolidityLexer.g4#L234
const regexIdentifier = new RegExp("^[a-zA-Z$_][a-zA-Z0-9$_]*$");
function verifyIdentifier(value: string): string {
    if (!value || !value.match(regexIdentifier)) {
        logger.throwArgumentError(`invalid identifier "${ value }"`, "value", value);
    }
    return value;
}

const regexParen = new RegExp("^([^)(]*)\\((.*)\\)([^)(]*)$");

function splitNesting(value: string): Array<any> {
    value = value.trim();

    let result = [];
    let accum = "";
    let depth = 0;
    for (let offset = 0; offset < value.length; offset++) {
        let c = value[offset];
        if (c === "," && depth === 0) {
            result.push(accum);
            accum = "";
        } else {
            accum += c;
            if (c === "(") {
                depth++;
            } else if (c === ")") {
                depth--;
                if (depth === -1) {
                    logger.throwArgumentError("unbalanced parenthesis", "value", value);
                }
            }
        }
    }
    if (accum) { result.push(accum); }

    return result;
}

