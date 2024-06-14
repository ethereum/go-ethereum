export declare class Semaphore {
    count: number;
    private _tasks;
    constructor(count: number);
    acquire(): Promise<() => void>;
    use<T>(f: () => Promise<T>): Promise<T>;
    private _sched;
}
export declare class Mutex extends Semaphore {
    constructor();
}
//# sourceMappingURL=index.d.ts.map