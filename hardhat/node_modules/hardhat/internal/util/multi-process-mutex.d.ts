export declare class MultiProcessMutex {
    private _mutexFilePath;
    private _mutexLifespanInMs;
    constructor(mutexName: string, maxMutexLifespanInMs?: number);
    use<T>(f: () => Promise<T>): Promise<T>;
    private _tryToAcquireMutex;
    private _executeFunctionAndReleaseMutex;
    private _isMutexFileTooOld;
    private _deleteMutexFile;
    private _waitMs;
}
//# sourceMappingURL=multi-process-mutex.d.ts.map