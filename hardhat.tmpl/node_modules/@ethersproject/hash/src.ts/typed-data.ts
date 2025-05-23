import { TypedDataDomain, TypedDataField } from "@ethersproject/abstract-signer";
import { getAddress } from "@ethersproject/address";
import { BigNumber, BigNumberish } from "@ethersproject/bignumber";
import { arrayify, BytesLike, hexConcat, hexlify, hexZeroPad, isHexString } from "@ethersproject/bytes";
import { keccak256 } from "@ethersproject/keccak256";
import { deepCopy, defineReadOnly, shallowCopy } from "@ethersproject/properties";

import { Logger } from "@ethersproject/logger";
import { version } from "./_version";
const logger = new Logger(version);

import { id } from "./id";

const padding = new Uint8Array(32);
padding.fill(0);

const NegativeOne: BigNumber = BigNumber.from(-1);
const Zero: BigNumber = BigNumber.from(0);
const One: BigNumber = BigNumber.from(1);
const MaxUint256: BigNumber = BigNumber.from("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");

function hexPadRight(value: BytesLike) {
    const bytes = arrayify(value);
    const padOffset = bytes.length % 32
    if (padOffset) {
        return hexConcat([ bytes, padding.slice(padOffset) ]);
    }
    return hexlify(bytes);
}

const hexTrue = hexZeroPad(One.toHexString(), 32);
const hexFalse = hexZeroPad(Zero.toHexString(), 32);

const domainFieldTypes: Record<string, string> = {
    name: "string",
    version: "string",
    chainId: "uint256",
    verifyingContract: "address",
    salt: "bytes32"
};

const domainFieldNames: Array<string> = [
    "name", "version", "chainId", "verifyingContract", "salt"
];

function checkString(key: string): (value: any) => string {
    return function (value: any){
        if (typeof(value) !== "string") {
            logger.throwArgumentError(`invalid domain value for ${ JSON.stringify(key) }`, `domain.${ key }`, value);
        }
        return value;
    }
}

const domainChecks: Record<string, (value: any) => any> = {
    name: checkString("name"),
    version: checkString("version"),
    chainId: function(value: any) {
        try {
            return BigNumber.from(value).toString()
        } catch (error) { }
        return logger.throwArgumentError(`invalid domain value for "chainId"`, "domain.chainId", value);
    },
    verifyingContract: function(value: any) {
        try {
            return getAddress(value).toLowerCase();
        } catch (error) { }
        return logger.throwArgumentError(`invalid domain value "verifyingContract"`, "domain.verifyingContract", value);
    },
    salt: function(value: any) {
        try {
            const bytes = arrayify(value);
            if (bytes.length !== 32) { throw new Error("bad length"); }
            return hexlify(bytes);
        } catch (error) { }
        return logger.throwArgumentError(`invalid domain value "salt"`, "domain.salt", value);
    }
}

function getBaseEncoder(type: string): (value: any) => string {
    // intXX and uintXX
    {
        const match = type.match(/^(u?)int(\d*)$/);
        if (match) {
            const signed = (match[1] === "");

            const width = parseInt(match[2] || "256");
            if (width % 8 !== 0 || width > 256 || (match[2] && match[2] !== String(width))) {
                logger.throwArgumentError("invalid numeric width", "type", type);
            }

            const boundsUpper = MaxUint256.mask(signed ? (width - 1): width);
            const boundsLower = signed ? boundsUpper.add(One).mul(NegativeOne): Zero;

            return function(value: BigNumberish) {
                const v = BigNumber.from(value);

                if (v.lt(boundsLower) || v.gt(boundsUpper)) {
                    logger.throwArgumentError(`value out-of-bounds for ${ type }`, "value", value);
                }

                return hexZeroPad(v.toTwos(256).toHexString(), 32);
            };
        }
    }

    // bytesXX
    {
        const match = type.match(/^bytes(\d+)$/);
        if (match) {
            const width = parseInt(match[1]);
            if (width === 0 || width > 32 || match[1] !== String(width)) {
                logger.throwArgumentError("invalid bytes width", "type", type);
            }

            return function(value: BytesLike) {
                const bytes = arrayify(value);
                if (bytes.length !== width) {
                    logger.throwArgumentError(`invalid length for ${ type }`, "value", value);
                }
                return hexPadRight(value);
            };
        }
    }

    switch (type) {
        case "address": return function(value: string) {
            return hexZeroPad(getAddress(value), 32);
        };
        case "bool": return function(value: boolean) {
            return ((!value) ? hexFalse: hexTrue);
        };
        case "bytes": return function(value: BytesLike) {
            return keccak256(value);
        };
        case "string": return function(value: string) {
            return id(value);
        };
    }

    return null;
}

function encodeType(name: string, fields: Array<TypedDataField>): string {
    return `${ name }(${ fields.map(({ name, type }) => (type + " " + name)).join(",") })`;
}

export class TypedDataEncoder {
    readonly primaryType: string;
    readonly types: Record<string, Array<TypedDataField>>;

    readonly _encoderCache: Record<string, (value: any) => string>;
    readonly _types: Record<string, string>;

    constructor(types: Record<string, Array<TypedDataField>>) {
        defineReadOnly(this, "types", Object.freeze(deepCopy(types)));

        defineReadOnly(this, "_encoderCache", { });
        defineReadOnly(this, "_types", { });

        // Link struct types to their direct child structs
        const links: Record<string, Record<string, boolean>> = { };

        // Link structs to structs which contain them as a child
        const parents: Record<string, Array<string>> = { };

        // Link all subtypes within a given struct
        const subtypes: Record<string, Record<string, boolean>> = { };

        Object.keys(types).forEach((type) => {
            links[type] = { };
            parents[type] = [ ];
            subtypes[type] = { }
        });

        for (const name in types) {

            const uniqueNames: Record<string, boolean> = { };

            types[name].forEach((field) => {

                // Check each field has a unique name
                if (uniqueNames[field.name]) {
                    logger.throwArgumentError(`duplicate variable name ${ JSON.stringify(field.name) } in ${ JSON.stringify(name) }`, "types", types);
                }
                uniqueNames[field.name] = true;

                // Get the base type (drop any array specifiers)
                const baseType = field.type.match(/^([^\x5b]*)(\x5b|$)/)[1];
                if (baseType === name) {
                    logger.throwArgumentError(`circular type reference to ${ JSON.stringify(baseType) }`, "types", types);
                }

                // Is this a base encoding type?
                const encoder = getBaseEncoder(baseType);
                if (encoder) { return ;}

                if (!parents[baseType]) {
                    logger.throwArgumentError(`unknown type ${ JSON.stringify(baseType) }`, "types", types);
                }

                // Add linkage
                parents[baseType].push(name);
                links[name][baseType] = true;
            });
        }

        // Deduce the primary type
        const primaryTypes = Object.keys(parents).filter((n) => (parents[n].length === 0));

        if (primaryTypes.length === 0) {
            logger.throwArgumentError("missing primary type", "types", types);
        } else if (primaryTypes.length > 1) {
            logger.throwArgumentError(`ambiguous primary types or unused types: ${ primaryTypes.map((t) => (JSON.stringify(t))).join(", ") }`, "types", types);
        }

        defineReadOnly(this, "primaryType", primaryTypes[0]);

        // Check for circular type references
        function checkCircular(type: string, found: Record<string, boolean>) {
            if (found[type]) {
                logger.throwArgumentError(`circular type reference to ${ JSON.stringify(type) }`, "types", types);
            }

            found[type] = true;

            Object.keys(links[type]).forEach((child) => {
                if (!parents[child]) { return; }

                // Recursively check children
                checkCircular(child, found);

                // Mark all ancestors as having this decendant
                Object.keys(found).forEach((subtype) => {
                    subtypes[subtype][child] = true;
                });
            });

            delete found[type];
        }
        checkCircular(this.primaryType, { });

        // Compute each fully describe type
        for (const name in subtypes) {
            const st = Object.keys(subtypes[name]);
            st.sort();
            this._types[name] = encodeType(name, types[name]) + st.map((t) => encodeType(t, types[t])).join("");
        }
    }

    getEncoder(type: string): (value: any) => string {
        let encoder = this._encoderCache[type];
        if (!encoder) {
            encoder = this._encoderCache[type] = this._getEncoder(type);
        }
        return encoder;
    }

    _getEncoder(type: string): (value: any) => string {

        // Basic encoder type (address, bool, uint256, etc)
        {
            const encoder = getBaseEncoder(type);
            if (encoder) { return encoder; }
        }

        // Array
        const match = type.match(/^(.*)(\x5b(\d*)\x5d)$/);
        if (match) {
            const subtype = match[1];
            const subEncoder = this.getEncoder(subtype);
            const length = parseInt(match[3]);
            return (value: Array<any>) => {
                if (length >= 0 && value.length !== length) {
                    logger.throwArgumentError("array length mismatch; expected length ${ arrayLength }", "value", value);
                }

                let result = value.map(subEncoder);
                if (this._types[subtype]) {
                    result = result.map(keccak256);
                }

                return keccak256(hexConcat(result));
            };
        }

        // Struct
        const fields = this.types[type];
        if (fields) {
            const encodedType = id(this._types[type]);
            return (value: Record<string, any>) => {
                const values = fields.map(({ name, type }) => {
                    const result = this.getEncoder(type)(value[name]);
                    if (this._types[type]) { return keccak256(result); }
                    return result;
                });
                values.unshift(encodedType);
                return hexConcat(values);
            }
        }

        return logger.throwArgumentError(`unknown type: ${ type }`, "type", type);
    }

    encodeType(name: string): string {
        const result = this._types[name];
        if (!result) {
            logger.throwArgumentError(`unknown type: ${ JSON.stringify(name) }`, "name", name);
        }
        return result;
    }

    encodeData(type: string, value: any): string {
        return this.getEncoder(type)(value);
    }

    hashStruct(name: string, value: Record<string, any>): string {
        return keccak256(this.encodeData(name, value));
    }

    encode(value: Record<string, any>): string {
        return this.encodeData(this.primaryType, value);
    }

    hash(value: Record<string, any>): string {
        return this.hashStruct(this.primaryType, value);
    }

    _visit(type: string, value: any, callback: (type: string, data: any) => any): any {
        // Basic encoder type (address, bool, uint256, etc)
        {
            const encoder = getBaseEncoder(type);
            if (encoder) { return callback(type, value); }
        }

        // Array
        const match = type.match(/^(.*)(\x5b(\d*)\x5d)$/);
        if (match) {
            const subtype = match[1];
            const length = parseInt(match[3]);
            if (length >= 0 && value.length !== length) {
                logger.throwArgumentError("array length mismatch; expected length ${ arrayLength }", "value", value);
            }
            return value.map((v: any) => this._visit(subtype, v, callback));
        }

        // Struct
        const fields = this.types[type];
        if (fields) {
            return fields.reduce((accum, { name, type }) => {
                accum[name] = this._visit(type, value[name], callback);
                return accum;
            }, <Record<string, any>>{});
        }

        return logger.throwArgumentError(`unknown type: ${ type }`, "type", type);
    }

    visit(value: Record<string, any>, callback: (type: string, data: any) => any): any {
        return this._visit(this.primaryType, value, callback);
    }

    static from(types: Record<string, Array<TypedDataField>>): TypedDataEncoder {
        return new TypedDataEncoder(types);
    }

    static getPrimaryType(types: Record<string, Array<TypedDataField>>): string {
        return TypedDataEncoder.from(types).primaryType;
    }

    static hashStruct(name: string, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string {
        return TypedDataEncoder.from(types).hashStruct(name, value);
    }

    static hashDomain(domain: TypedDataDomain): string {
        const domainFields: Array<TypedDataField> = [ ];
        for (const name in domain) {
            const type = domainFieldTypes[name];
            if (!type) {
                logger.throwArgumentError(`invalid typed-data domain key: ${ JSON.stringify(name) }`, "domain", domain);
            }
            domainFields.push({ name, type });
        }

        domainFields.sort((a, b) => {
            return domainFieldNames.indexOf(a.name) - domainFieldNames.indexOf(b.name);
        });

        return TypedDataEncoder.hashStruct("EIP712Domain", { EIP712Domain: domainFields }, domain);
    }

    static encode(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string {
        return hexConcat([
            "0x1901",
            TypedDataEncoder.hashDomain(domain),
            TypedDataEncoder.from(types).hash(value)
        ]);
    }

    static hash(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string {
        return keccak256(TypedDataEncoder.encode(domain, types, value));
    }

    // Replaces all address types with ENS names with their looked up address
    static async resolveNames(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>, resolveName: (name: string) => Promise<string>): Promise<{ domain: TypedDataDomain, value: any }> {
        // Make a copy to isolate it from the object passed in
        domain = shallowCopy(domain);

        // Look up all ENS names
        const ensCache: Record<string, string> = { };

        // Do we need to look up the domain's verifyingContract?
        if (domain.verifyingContract && !isHexString(domain.verifyingContract, 20)) {
            ensCache[domain.verifyingContract] = "0x";
        }

        // We are going to use the encoder to visit all the base values
        const encoder = TypedDataEncoder.from(types);

        // Get a list of all the addresses
        encoder.visit(value, (type: string, value: any) => {
            if (type === "address" && !isHexString(value, 20)) {
                ensCache[value] = "0x";
            }
            return value;
        });

        // Lookup each name
        for (const name in ensCache) {
            ensCache[name] = await resolveName(name);
        }

        // Replace the domain verifyingContract if needed
        if (domain.verifyingContract && ensCache[domain.verifyingContract]) {
            domain.verifyingContract = ensCache[domain.verifyingContract];
        }

        // Replace all ENS names with their address
        value = encoder.visit(value, (type: string, value: any) => {
            if (type === "address" && ensCache[value]) { return ensCache[value]; }
            return value;
        });

        return { domain, value };
    }

    static getPayload(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): any {
        // Validate the domain fields
        TypedDataEncoder.hashDomain(domain);

        // Derive the EIP712Domain Struct reference type
        const domainValues: Record<string, any> = { };
        const domainTypes: Array<{ name: string, type:string }> = [ ];

        domainFieldNames.forEach((name) => {
            const value = (<any>domain)[name];
            if (value == null) { return; }
            domainValues[name] = domainChecks[name](value);
            domainTypes.push({ name, type: domainFieldTypes[name] });
        });

        const encoder = TypedDataEncoder.from(types);

        const typesWithDomain = shallowCopy(types);
        if (typesWithDomain.EIP712Domain) {
            logger.throwArgumentError("types must not contain EIP712Domain type", "types.EIP712Domain", types);
        } else {
            typesWithDomain.EIP712Domain = domainTypes;
        }

        // Validate the data structures and types
        encoder.encode(value);

        return {
            types: typesWithDomain,
            domain: domainValues,
            primaryType: encoder.primaryType,
            message: encoder.visit(value, (type: string, value: any) => {

                // bytes
                if (type.match(/^bytes(\d*)/)) {
                    return hexlify(arrayify(value));
                }

                // uint or int
                if (type.match(/^u?int/)) {
                    return BigNumber.from(value).toString();
                }

                switch (type) {
                    case "address":
                        return value.toLowerCase();
                    case "bool":
                        return !!value;
                    case "string":
                        if (typeof(value) !== "string") {
                            logger.throwArgumentError(`invalid string`, "value", value);
                        }
                        return value;
                }

                return logger.throwArgumentError("unsupported type", "type", type);
            })
        };
    }
}

