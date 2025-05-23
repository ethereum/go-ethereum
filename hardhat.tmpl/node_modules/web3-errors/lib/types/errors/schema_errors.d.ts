import { BaseWeb3Error } from '../web3_error_base.js';
export declare class SchemaFormatError extends BaseWeb3Error {
    type: string;
    code: number;
    constructor(type: string);
    toJSON(): {
        type: string;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
//# sourceMappingURL=schema_errors.d.ts.map