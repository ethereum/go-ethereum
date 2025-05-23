import { Web3ValidationErrorObject } from 'web3-types';
import { ValidationSchemaInput, Web3ValidationOptions } from './types.js';
export declare class Web3Validator {
    private readonly _validator;
    constructor();
    validateJSONSchema(schema: object, data: object, options?: Web3ValidationOptions): Web3ValidationErrorObject[] | undefined;
    validate(schema: ValidationSchemaInput, data: ReadonlyArray<unknown>, options?: Web3ValidationOptions): Web3ValidationErrorObject[] | undefined;
}
