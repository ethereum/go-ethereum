import { EventEmitter } from 'web3-utils';
export type Web3EventMap = Record<string, unknown>;
export type Web3EventKey<T extends Web3EventMap> = string & keyof T;
export type Web3EventCallback<T> = (params: T) => void | Promise<void>;
export interface Web3Emitter<T extends Web3EventMap> {
    on<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    once<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    off<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    emit<K extends Web3EventKey<T>>(eventName: K, params: T[K]): void;
}
export declare class Web3EventEmitter<T extends Web3EventMap> implements Web3Emitter<T> {
    private readonly _emitter;
    on<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    once<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    off<K extends Web3EventKey<T>>(eventName: K, fn: Web3EventCallback<T[K]>): void;
    emit<K extends Web3EventKey<T>>(eventName: K, params: T[K]): void;
    listenerCount<K extends Web3EventKey<T>>(eventName: K): number;
    listeners<K extends Web3EventKey<T>>(eventName: K): ((...args: any[]) => void)[];
    eventNames(): (string | symbol)[];
    removeAllListeners(): EventEmitter;
    setMaxListenerWarningThreshold(maxListenersWarningThreshold: number): void;
    getMaxListeners(): number;
}
