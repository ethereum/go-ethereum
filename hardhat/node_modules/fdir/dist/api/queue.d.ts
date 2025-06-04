import { WalkerState } from "../types";
type OnQueueEmptyCallback = (error: Error | null, output: WalkerState) => void;
/**
 * This is a custom stateless queue to track concurrent async fs calls.
 * It increments a counter whenever a call is queued and decrements it
 * as soon as it completes. When the counter hits 0, it calls onQueueEmpty.
 */
export declare class Queue {
    private onQueueEmpty?;
    count: number;
    constructor(onQueueEmpty?: OnQueueEmptyCallback | undefined);
    enqueue(): number;
    dequeue(error: Error | null, output: WalkerState): void;
}
export {};
