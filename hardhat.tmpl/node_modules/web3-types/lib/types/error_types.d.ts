export interface Web3Error extends Error {
    readonly name: string;
    readonly code: number;
    readonly stack?: string;
}
export type Web3ValidationErrorObject<K extends string = string, P = Record<string, any>, S = unknown> = {
    keyword: K;
    instancePath: string;
    schemaPath: string;
    params: P;
    propertyName?: string;
    message?: string;
    schema?: S;
    data?: unknown;
};
//# sourceMappingURL=error_types.d.ts.map