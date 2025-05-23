import { Web3ValidationErrorObject } from 'web3-types';
import { Json, JsonSchema } from './types.js';
export declare class Validator {
    private static validatorInstance?;
    static factory(): Validator;
    validate(schema: JsonSchema, data: Json, options?: {
        silent?: boolean;
    }): Web3ValidationErrorObject<string, Record<string, any>, unknown>[] | undefined;
    private convertErrors;
}
