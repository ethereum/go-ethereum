/** A simple queue that holds promises. */
export declare class PromiseBuffer<T> {
    protected _limit?: number | undefined;
    /** Internal set of queued Promises */
    private readonly _buffer;
    constructor(_limit?: number | undefined);
    /**
     * Says if the buffer is ready to take more requests
     */
    isReady(): boolean;
    /**
     * Add a promise to the queue.
     *
     * @param task Can be any PromiseLike<T>
     * @returns The original promise.
     */
    add(task: PromiseLike<T>): PromiseLike<T>;
    /**
     * Remove a promise to the queue.
     *
     * @param task Can be any PromiseLike<T>
     * @returns Removed promise.
     */
    remove(task: PromiseLike<T>): PromiseLike<T>;
    /**
     * This function returns the number of unresolved promises in the queue.
     */
    length(): number;
    /**
     * This will drain the whole queue, returns true if queue is empty or drained.
     * If timeout is provided and the queue takes longer to drain, the promise still resolves but with false.
     *
     * @param timeout Number in ms to wait until it resolves with false.
     */
    drain(timeout?: number): PromiseLike<boolean>;
}
//# sourceMappingURL=promisebuffer.d.ts.map