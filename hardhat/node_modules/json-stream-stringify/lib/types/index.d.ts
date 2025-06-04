import { Readable } from 'stream';

interface Item {
    read(size?: number): Promise<void> | void;
    depth?: number;
    value?: any;
    indent?: string;
    path?: (string | number)[];
    type?: string;
}
declare enum ReadState {
    Inactive = 0,
    Reading = 1,
    ReadMore = 2,
    Consumed = 3
}
declare class JsonStreamStringify extends Readable {
    private cycle;
    private bufferSize;
    item?: Item;
    indent?: string;
    root: Item;
    include: string[];
    replacer: Function;
    visited: [] | WeakMap<any, string[]>;
    constructor(input: any, replacer?: Function | any[] | undefined, spaces?: number | string | undefined, cycle?: boolean, bufferSize?: number);
    setItem(value: any, parent: Item, key?: string | number): void;
    setReadableStringItem(input: Readable, parent: Item): void;
    setReadableObjectItem(input: Readable, parent: Item): void;
    setPromiseItem(input: Promise<any>, parent: Item, key: any): void;
    setArrayItem(input: any[], parent: any): void;
    unvisit(item: any): void;
    objectItem?: any;
    setObjectItem(input: Record<any, any>, parent?: any): void;
    buffer: string;
    bufferLength: number;
    pushCalled: boolean;
    readSize: number;
    /** if set, this string will be prepended to the next _push call, if the call output is not empty, and set to undefined */
    prePush?: string;
    private _push;
    readState: ReadState;
    _read(size?: number): Promise<void>;
    private cleanup;
    destroy(error?: Error): this;
}

export { JsonStreamStringify };
