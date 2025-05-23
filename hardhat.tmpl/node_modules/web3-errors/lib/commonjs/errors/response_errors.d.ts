import { JsonRpcPayload, JsonRpcResponse } from 'web3-types';
import { BaseWeb3Error } from '../web3_error_base.js';
export declare class ResponseError<ErrorType = unknown, RequestType = unknown> extends BaseWeb3Error {
    code: number;
    data?: ErrorType | ErrorType[];
    request?: JsonRpcPayload<RequestType>;
    statusCode?: number;
    constructor(response: JsonRpcResponse<unknown, ErrorType>, message?: string, request?: JsonRpcPayload<RequestType>, statusCode?: number);
    toJSON(): {
        data: ErrorType | ErrorType[] | undefined;
        request: JsonRpcPayload<RequestType> | undefined;
        statusCode: number | undefined;
        name: string;
        code: number;
        message: string;
        cause: Error | undefined;
        innerError: Error | undefined;
    };
}
export declare class InvalidResponseError<ErrorType = unknown, RequestType = unknown> extends ResponseError<ErrorType, RequestType> {
    constructor(result: JsonRpcResponse<unknown, ErrorType>, request?: JsonRpcPayload<RequestType>);
}
