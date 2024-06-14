import { Dictionary } from 'ts-essentials';
import { EvmOutputType, EvmType, StructType } from './parseEvmType';
export interface AbiParameter {
    name: string;
    type: EvmType;
}
export interface AbiOutputParameter {
    name: string;
    type: EvmOutputType;
}
export type Named<T> = {
    name: string;
    values: T;
};
export type StateMutability = 'pure' | 'view' | 'nonpayable' | 'payable';
export interface FunctionDocumentation {
    author?: string;
    details?: string;
    notice?: string;
    params?: {
        [paramName: string]: string;
    };
    return?: string;
}
export interface FunctionDeclaration {
    name: string;
    stateMutability: StateMutability;
    inputs: AbiParameter[];
    outputs: AbiOutputParameter[];
    documentation?: FunctionDocumentation | undefined;
}
export interface FunctionWithoutOutputDeclaration extends FunctionDeclaration {
    outputs: [];
}
export interface FunctionWithoutInputDeclaration extends FunctionDeclaration {
    inputs: [];
}
export interface Contract {
    name: string;
    rawName: string;
    path: string[];
    fallback?: FunctionWithoutInputDeclaration | undefined;
    constructor: FunctionWithoutOutputDeclaration[];
    functions: Dictionary<FunctionDeclaration[]>;
    events: Dictionary<EventDeclaration[]>;
    structs: Dictionary<StructType[]>;
    documentation?: {
        author?: string;
        details?: string;
        notice?: string;
    } | undefined;
}
export interface RawAbiParameter {
    name: string;
    type: string;
    internalType?: string;
    components?: RawAbiParameter[];
}
export interface RawAbiDefinition {
    name: string;
    constant: boolean;
    payable: boolean;
    stateMutability?: StateMutability;
    inputs: RawAbiParameter[];
    outputs: RawAbiParameter[];
    type: string;
}
export interface EventDeclaration {
    name: string;
    isAnonymous: boolean;
    inputs: EventArgDeclaration[];
}
export interface EventArgDeclaration {
    isIndexed: boolean;
    name?: string | undefined;
    type: EvmType;
}
export interface RawEventAbiDefinition {
    type: 'event';
    anonymous?: boolean;
    name: string;
    inputs: RawEventArgAbiDefinition[];
}
export interface RawEventArgAbiDefinition {
    indexed: boolean;
    name: string;
    type: string;
}
export interface BytecodeLinkReference {
    reference: string;
    name?: string;
}
export interface BytecodeWithLinkReferences {
    bytecode: string;
    linkReferences?: BytecodeLinkReference[];
}
export interface DocumentationResult {
    author?: string;
    details?: string;
    notice?: string;
    title?: string;
    methods?: {
        [methodName: string]: FunctionDocumentation;
    };
}
export declare function parseContractPath(path: string): {
    name: string;
    rawName: string;
    path: string[];
};
export declare function parse(abi: RawAbiDefinition[], path: string, documentation?: DocumentationResult): Contract;
export declare function parseEvent(abiPiece: RawEventAbiDefinition, registerStruct: (struct: StructType) => void): EventDeclaration;
export declare function getFunctionDocumentation(abiPiece: RawAbiDefinition, documentation?: DocumentationResult): FunctionDocumentation | undefined;
export declare function extractAbi(rawJson: string): RawAbiDefinition[];
export declare function extractBytecode(rawContents: string): BytecodeWithLinkReferences | undefined;
export declare function extractDocumentation(rawContents: string): DocumentationResult | undefined;
export declare function ensure0xPrefix(hexString: string): string;
export declare function isConstant(fn: FunctionDeclaration): boolean;
export declare function isConstantFn(fn: FunctionDeclaration): boolean;
