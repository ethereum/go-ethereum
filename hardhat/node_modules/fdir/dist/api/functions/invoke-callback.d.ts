import { Output, ResultCallback, WalkerState, Options } from "../../types";
export type InvokeCallbackFunction<TOutput extends Output> = (state: WalkerState, error: Error | null, callback?: ResultCallback<TOutput>) => null | TOutput;
export declare function build<TOutput extends Output>(options: Options, isSynchronous: boolean): InvokeCallbackFunction<TOutput>;
