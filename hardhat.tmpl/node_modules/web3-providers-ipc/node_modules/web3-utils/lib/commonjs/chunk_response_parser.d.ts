import { JsonRpcResponse } from 'web3-types';
import { EventEmitter } from 'eventemitter3';
export declare class ChunkResponseParser {
    private lastChunk;
    private lastChunkTimeout;
    private _clearQueues;
    private readonly eventEmitter;
    private readonly autoReconnect;
    private readonly chunkTimeout;
    constructor(eventEmitter: EventEmitter, autoReconnect: boolean);
    private clearQueues;
    onError(clearQueues?: () => void): void;
    parseResponse(data: string): JsonRpcResponse[];
}
