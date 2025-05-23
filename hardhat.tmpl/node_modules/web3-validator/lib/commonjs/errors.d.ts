import { BaseWeb3Error } from 'web3-errors';
import { Web3ValidationErrorObject } from 'web3-types';
export declare class Web3ValidatorError extends BaseWeb3Error {
    code: number;
    readonly errors: Web3ValidationErrorObject[];
    constructor(errors: Web3ValidationErrorObject[]);
    private _compileErrors;
}
