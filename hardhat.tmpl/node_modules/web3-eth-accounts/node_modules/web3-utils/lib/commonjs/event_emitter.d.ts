import EventEmitter3 from 'eventemitter3';
/**
 * This class copy the behavior of Node.js EventEmitter class.
 * It is used to provide the same interface for the browser environment.
 */
export declare class EventEmitter extends EventEmitter3 {
    private maxListeners;
    setMaxListeners(maxListeners: number): this;
    getMaxListeners(): number;
}
