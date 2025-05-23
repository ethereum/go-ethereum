import { AbiParameter, EventArgDeclaration, EventDeclaration, FunctionDeclaration } from '../parser/abiParser';
export declare function getFullSignatureAsSymbolForEvent(event: EventDeclaration): string;
export declare function getFullSignatureForEvent(event: EventDeclaration): string;
export declare function getIndexedSignatureForEvent(event: EventDeclaration): string;
export declare function getArgumentForSignature(argument: EventArgDeclaration | AbiParameter): string;
export declare function getSignatureForFn(fn: FunctionDeclaration): string;
