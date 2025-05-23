import { HexString, JsonRpcPayload, JsonRpcResponse, Transaction, Web3APIMethod, Web3APIReturnType } from 'web3-types';
import { Schema } from 'web3-validator';
export type TransactionTypeParser = (transaction: Transaction) => HexString | undefined;
export interface Method {
    name: string;
    call: string;
}
export interface ExtensionObject {
    property?: string;
    methods: Method[];
}
export interface RequestManagerMiddleware<API> {
    processRequest<ParamType = unknown[]>(request: JsonRpcPayload<ParamType>, options?: {
        [key: string]: unknown;
    }): Promise<JsonRpcPayload<ParamType>>;
    processResponse<AnotherMethod extends Web3APIMethod<API>, ResponseType = Web3APIReturnType<API, AnotherMethod>>(response: JsonRpcResponse<ResponseType>, options?: {
        [key: string]: unknown;
    }): Promise<JsonRpcResponse<ResponseType>>;
}
export type CustomTransactionSchema = {
    type: string;
    properties: Record<string, Schema>;
};
