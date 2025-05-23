import { ResultCallback, Options } from "../types";
import { Output } from "../types";
export declare class Walker<TOutput extends Output> {
    private readonly root;
    private readonly isSynchronous;
    private readonly state;
    private readonly joinPath;
    private readonly pushDirectory;
    private readonly pushFile;
    private readonly getArray;
    private readonly groupFiles;
    private readonly resolveSymlink;
    private readonly walkDirectory;
    private readonly callbackInvoker;
    constructor(root: string, options: Options, callback?: ResultCallback<TOutput>);
    start(): TOutput | null;
    private walk;
}
