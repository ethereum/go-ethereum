import { AbiParameter } from 'web3-types';
import { ZodIssueBase } from 'zod';
export declare type ValidInputTypes = Uint8Array | bigint | string | number | boolean;
export declare type EthBaseTypes = 'bool' | 'bytes' | 'string' | 'uint' | 'int' | 'address' | 'tuple';
export declare type EthBaseTypesWithMeta = `string${string}` | `string${string}[${number}]` | `bytes${string}` | `bytes${string}[${number}]` | `address[${number}]` | `bool[${number}]` | `int${string}` | `int${string}[${number}]` | `uint${string}` | `uint${string}[${number}]` | `tuple[]` | `tuple[${number}]`;
export declare type EthExtendedTypes = 'hex' | 'number' | 'blockNumber' | 'blockNumberOrTag' | 'filter' | 'bloom';
export declare type FullValidationSchema = ReadonlyArray<AbiParameter>;
export declare type ShortValidationSchema = ReadonlyArray<string | EthBaseTypes | EthExtendedTypes | EthBaseTypesWithMeta | ShortValidationSchema>;
export declare type ValidationSchemaInput = FullValidationSchema | ShortValidationSchema;
export declare type Web3ValidationOptions = {
    readonly silent: boolean;
};
export declare type Json = string | number | boolean | Array<Json> | {
    [id: string]: Json;
};
export declare type ValidationError = ZodIssueBase;
export interface Validate {
    (value: Json): boolean;
    errors?: ValidationError[];
}
export declare type Schema = {
    $schema?: string;
    $vocabulary?: string;
    id?: string;
    $id?: string;
    $anchor?: string;
    $ref?: string;
    definitions?: {
        [id: string]: Schema;
    };
    $defs?: {
        [id: string]: Schema;
    };
    $recursiveRef?: string;
    $recursiveAnchor?: boolean;
    type?: string | Array<string>;
    required?: Array<string> | boolean;
    default?: Json;
    enum?: Array<Json>;
    const?: Json;
    not?: Schema;
    allOf?: Array<Schema>;
    anyOf?: Array<Schema>;
    oneOf?: Array<Schema>;
    if?: Schema;
    then?: Schema;
    else?: Schema;
    maximum?: number;
    minimum?: number;
    exclusiveMaximum?: number | boolean;
    exclusiveMinimum?: number | boolean;
    multipleOf?: number;
    divisibleBy?: number;
    maxItems?: number;
    minItems?: number;
    additionalItems?: Schema;
    contains?: Schema;
    minContains?: number;
    maxContains?: number;
    uniqueItems?: boolean;
    maxLength?: number;
    minLength?: number;
    format?: string;
    pattern?: string;
    contentEncoding?: string;
    contentMediaType?: string;
    contentSchema?: Schema;
    properties?: {
        [id: string]: Schema;
    };
    maxProperties?: number;
    minProperties?: number;
    additionalProperties?: Schema;
    patternProperties?: {
        [pattern: string]: Schema;
    };
    propertyNames?: Schema;
    dependencies?: {
        [id: string]: Array<string> | Schema;
    };
    dependentRequired?: {
        [id: string]: Array<string>;
    };
    dependentSchemas?: {
        [id: string]: Schema;
    };
    unevaluatedProperties?: Schema;
    unevaluatedItems?: Schema;
    title?: string;
    description?: string;
    deprecated?: boolean;
    readOnly?: boolean;
    writeOnly?: boolean;
    examples?: Array<Json>;
    $comment?: string;
    discriminator?: {
        propertyName: string;
        mapping?: {
            [value: string]: string;
        };
    };
    readonly eth?: string;
    items?: Schema | Schema[];
};
export declare type JsonSchema = Schema;
