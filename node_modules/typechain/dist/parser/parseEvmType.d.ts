export type EvmType = BooleanType | IntegerType | UnsignedIntegerType | StringType | BytesType | DynamicBytesType | AddressType | ArrayType | TupleType | UnknownType;
/**
 * Like EvmType but with void
 */
export type EvmOutputType = EvmType | VoidType;
export type StructType = ArrayType | TupleType;
export type BooleanType = {
    type: 'boolean';
    originalType: string;
};
export type IntegerType = {
    type: 'integer';
    bits: number;
    originalType: string;
};
export type UnsignedIntegerType = {
    type: 'uinteger';
    bits: number;
    originalType: string;
};
export type StringType = {
    type: 'string';
    originalType: string;
};
export type BytesType = {
    type: 'bytes';
    size: number;
    originalType: string;
};
export type DynamicBytesType = {
    type: 'dynamic-bytes';
    originalType: string;
};
export type AddressType = {
    type: 'address';
    originalType: string;
};
export type ArrayType = {
    type: 'array';
    itemType: EvmType;
    size?: number;
    originalType: string;
    structName?: StructName;
};
export type TupleType = {
    type: 'tuple';
    components: EvmSymbol[];
    originalType: string;
    structName?: StructName;
};
export type VoidType = {
    type: 'void';
};
export type UnknownType = {
    type: 'unknown';
    originalType: string;
};
export declare class StructName {
    readonly identifier: string;
    readonly namespace?: string;
    constructor(_identifier: string, _namespace?: string);
    toString(): string;
    merge(other: Partial<StructName>): StructName;
}
export type EvmSymbol = {
    type: EvmType;
    name: string;
};
export declare function parseEvmType(rawType: string, components?: EvmSymbol[], internalType?: string): EvmType;
/** @internal */
export declare function extractStructNameIfAvailable(internalType: string | undefined): StructName | undefined;
