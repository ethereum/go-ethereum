import { TypedDataDomain, TypedDataField } from "@ethersproject/abstract-signer";
export declare class TypedDataEncoder {
    readonly primaryType: string;
    readonly types: Record<string, Array<TypedDataField>>;
    readonly _encoderCache: Record<string, (value: any) => string>;
    readonly _types: Record<string, string>;
    constructor(types: Record<string, Array<TypedDataField>>);
    getEncoder(type: string): (value: any) => string;
    _getEncoder(type: string): (value: any) => string;
    encodeType(name: string): string;
    encodeData(type: string, value: any): string;
    hashStruct(name: string, value: Record<string, any>): string;
    encode(value: Record<string, any>): string;
    hash(value: Record<string, any>): string;
    _visit(type: string, value: any, callback: (type: string, data: any) => any): any;
    visit(value: Record<string, any>, callback: (type: string, data: any) => any): any;
    static from(types: Record<string, Array<TypedDataField>>): TypedDataEncoder;
    static getPrimaryType(types: Record<string, Array<TypedDataField>>): string;
    static hashStruct(name: string, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string;
    static hashDomain(domain: TypedDataDomain): string;
    static encode(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string;
    static hash(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): string;
    static resolveNames(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>, resolveName: (name: string) => Promise<string>): Promise<{
        domain: TypedDataDomain;
        value: any;
    }>;
    static getPayload(domain: TypedDataDomain, types: Record<string, Array<TypedDataField>>, value: Record<string, any>): any;
}
//# sourceMappingURL=typed-data.d.ts.map