import * as t from "io-ts";
/**
 * This function validates a list of params, throwing InvalidArgumentsError
 * if the validation fails, and returning their already-parsed types if
 * the validation succeeds.
 *
 * TODO: The type can probably be improved, removing the anys
 */
export declare function validateParams<TypesT extends ReadonlyArray<t.Type<any, any, any>>>(params: any[], ...types: TypesT): {
    [i in keyof TypesT]: TypesT[i] extends t.Type<infer TypeT, any, any> ? TypeT : never;
};
//# sourceMappingURL=validation.d.ts.map