import { bytesToUnprefixedHex } from './bytes.js'

import type { BatchDBOp, DB, DBObject } from './db.js'

export class MapDB<
  TKey extends Uint8Array | string | number,
  TValue extends Uint8Array | string | DBObject
> implements DB<TKey, TValue>
{
  _database: Map<TKey, TValue>

  constructor(database?: Map<TKey, TValue>) {
    this._database = database ?? new Map<TKey, TValue>()
  }

  async get(key: TKey): Promise<TValue | undefined> {
    const dbKey = key instanceof Uint8Array ? bytesToUnprefixedHex(key) : key.toString()
    return this._database.get(dbKey as TKey)
  }

  async put(key: TKey, val: TValue): Promise<void> {
    const dbKey = key instanceof Uint8Array ? bytesToUnprefixedHex(key) : key.toString()
    this._database.set(dbKey as TKey, val)
  }

  async del(key: TKey): Promise<void> {
    const dbKey = key instanceof Uint8Array ? bytesToUnprefixedHex(key) : key.toString()
    this._database.delete(dbKey as TKey)
  }

  async batch(opStack: BatchDBOp<TKey, TValue>[]): Promise<void> {
    for (const op of opStack) {
      if (op.type === 'del') {
        await this.del(op.key)
      }

      if (op.type === 'put') {
        await this.put(op.key, op.value)
      }
    }
  }

  /**
   * Note that the returned shallow copy will share the underlying database with the original
   *
   * @returns DB
   */
  shallowCopy(): DB<TKey, TValue> {
    return new MapDB<TKey, TValue>(this._database)
  }

  open() {
    return Promise.resolve()
  }
}
