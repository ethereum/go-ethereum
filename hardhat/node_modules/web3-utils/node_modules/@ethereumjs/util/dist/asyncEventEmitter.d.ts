/**
 * Ported to Typescript from original implementation below:
 * https://github.com/ahultgren/async-eventemitter -- MIT licensed
 *
 * Type Definitions based on work by: patarapolw <https://github.com/patarapolw> -- MIT licensed
 * that was contributed to Definitely Typed below:
 * https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/async-eventemitter
 */
/// <reference types="node" />
import { EventEmitter } from 'events';
declare type AsyncListener<T, R> = ((data: T, callback?: (result?: R) => void) => Promise<R>) | ((data: T, callback?: (result?: R) => void) => void);
export interface EventMap {
    [event: string]: AsyncListener<any, any>;
}
export declare class AsyncEventEmitter<T extends EventMap> extends EventEmitter {
    emit<E extends keyof T>(event: E & string, ...args: Parameters<T[E]>): boolean;
    once<E extends keyof T>(event: E & string, listener: T[E]): this;
    first<E extends keyof T>(event: E & string, listener: T[E]): this;
    before<E extends keyof T>(event: E & string, target: T[E], listener: T[E]): this;
    after<E extends keyof T>(event: E & string, target: T[E], listener: T[E]): this;
    private beforeOrAfter;
    on<E extends keyof T>(event: E & string, listener: T[E]): this;
    addListener<E extends keyof T>(event: E & string, listener: T[E]): this;
    prependListener<E extends keyof T>(event: E & string, listener: T[E]): this;
    prependOnceListener<E extends keyof T>(event: E & string, listener: T[E]): this;
    removeAllListeners(event?: keyof T & string): this;
    removeListener<E extends keyof T>(event: E & string, listener: T[E]): this;
    eventNames(): Array<keyof T & string>;
    listeners<E extends keyof T>(event: E & string): Array<T[E]>;
    listenerCount(event: keyof T & string): number;
    getMaxListeners(): number;
    setMaxListeners(maxListeners: number): this;
}
export {};
//# sourceMappingURL=asyncEventEmitter.d.ts.map