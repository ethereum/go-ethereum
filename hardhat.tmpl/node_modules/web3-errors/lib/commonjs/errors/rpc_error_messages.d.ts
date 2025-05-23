/**
 * A template string for a generic Rpc Error. The `*code*` will be replaced with the code number.
 * Note: consider in next version that a spelling mistake could be corrected for `occured` and the value could be:
 * 	`An Rpc error has occurred with a code of *code*`
 */
export declare const genericRpcErrorMessageTemplate = "An Rpc error has occured with a code of *code*";
export declare const RpcErrorMessages: {
    [key: number | string]: {
        name?: string;
        message: string;
        description?: string;
    };
};
