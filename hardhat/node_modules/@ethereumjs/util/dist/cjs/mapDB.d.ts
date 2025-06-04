import type { BatchDBOp, DB, DBObject } from './db.js';
export declare class MapDB<TKey extends Uint8Array | string | number, TValue extends Uint8Array | string | DBObject> implements DB<TKey, TValue> {
    _database: Map<TKey, TValue>;
    constructor(database?: Map<TKey, TValue>);
    get(key: TKey): Promise<TValue | undefined>;
    put(key: TKey, val: TValue): Promise<void>;
    del(key: TKey): Promise<void>;
    batch(opStack: BatchDBOp<TKey, TValue>[]): Promise<void>;
    /**
     * Note that the returned shallow copy will share the underlying database with the original
     *
     * @returns DB
     */
    shallowCopy(): DB<TKey, TValue>;
    open(): Promise<void>;
}
//# sourceMappingURL=mapDB.d.ts.map