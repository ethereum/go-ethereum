import { Output, Options, ResultCallback } from "../types";
export declare function promise<TOutput extends Output>(root: string, options: Options): Promise<TOutput>;
export declare function callback<TOutput extends Output>(root: string, options: Options, callback: ResultCallback<TOutput>): void;
