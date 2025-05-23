export type Schema = ObjectSchema | ArraySchema | StringSchema | NumberSchema | IntegerSchema | BooleanSchema | NullSchema;
export interface BaseSchema {
    type?: string | undefined;
    $id?: string | undefined;
    id?: string | undefined;
    $schema?: string | undefined;
    $ref?: string | undefined;
    $anchor?: string | undefined;
    $defs?: {
        [key: string]: BaseSchema;
    } | undefined;
    definitions?: {
        [key: string]: BaseSchema;
    } | undefined;
    $comment?: string | undefined;
    title?: string | undefined;
    description?: string | undefined;
    default?: unknown | undefined;
    examples?: unknown[] | undefined;
    readOnly?: boolean | undefined;
    writeOnly?: boolean | undefined;
    deprecated?: boolean | undefined;
    allOf?: BaseSchema[] | undefined;
    anyOf?: BaseSchema[] | undefined;
    oneOf?: BaseSchema[] | undefined;
    not?: BaseSchema | undefined;
    if?: BaseSchema | undefined;
    then?: BaseSchema | undefined;
    else?: BaseSchema | undefined;
    enum?: Array<string | number | boolean | null> | undefined;
    const?: string | number | boolean | null | undefined;
    [k: string]: unknown;
    /** A special key used as an intermediate representation of extends-style relationships. Omitted as a $ref with additional properties. */
    _prefault?: unknown | undefined;
}
export interface ObjectSchema extends BaseSchema {
    type: "object";
    properties?: {
        [key: string]: BaseSchema;
    } | undefined;
    patternProperties?: {
        [key: string]: BaseSchema;
    } | undefined;
    additionalProperties?: BaseSchema | boolean | undefined;
    required?: string[] | undefined;
    dependentRequired?: {
        [key: string]: string[];
    } | undefined;
    propertyNames?: BaseSchema | undefined;
    minProperties?: number | undefined;
    maxProperties?: number | undefined;
    unevaluatedProperties?: BaseSchema | boolean | undefined;
    dependentSchemas?: {
        [key: string]: BaseSchema;
    } | undefined;
}
export interface ArraySchema extends BaseSchema {
    type: "array";
    items?: BaseSchema | BaseSchema[] | undefined;
    prefixItems?: BaseSchema[] | undefined;
    additionalItems?: BaseSchema | boolean;
    contains?: BaseSchema | undefined;
    minItems?: number | undefined;
    maxItems?: number | undefined;
    minContains?: number | undefined;
    maxContains?: number | undefined;
    uniqueItems?: boolean | undefined;
    unevaluatedItems?: BaseSchema | boolean | undefined;
}
export interface StringSchema extends BaseSchema {
    type: "string";
    minLength?: number | undefined;
    maxLength?: number | undefined;
    pattern?: string | undefined;
    format?: string | undefined;
    contentEncoding?: string | undefined;
    contentMediaType?: string | undefined;
}
export interface NumberSchema extends BaseSchema {
    type: "number";
    minimum?: number | undefined;
    maximum?: number | undefined;
    exclusiveMinimum?: number | undefined;
    exclusiveMaximum?: number | undefined;
    multipleOf?: number | undefined;
}
export interface IntegerSchema extends BaseSchema {
    type: "integer";
    minimum?: number | undefined;
    maximum?: number | undefined;
    exclusiveMinimum?: number | undefined;
    exclusiveMaximum?: number | undefined;
    multipleOf?: number | undefined;
}
export interface BooleanSchema extends BaseSchema {
    type: "boolean";
}
export interface NullSchema extends BaseSchema {
    type: "null";
}
