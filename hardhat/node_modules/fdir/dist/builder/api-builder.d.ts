import { Options, Output, ResultCallback } from "../types";
export declare class APIBuilder<TReturnType extends Output> {
    private readonly root;
    private readonly options;
    constructor(root: string, options: Options);
    withPromise(): Promise<TReturnType>;
    withCallback(cb: ResultCallback<TReturnType>): void;
    sync(): TReturnType;
}
