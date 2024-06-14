"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.MapDB = void 0;
const bytes_js_1 = require("./bytes.js");
class MapDB {
    constructor(database) {
        this._database = database ?? new Map();
    }
    async get(key) {
        const dbKey = key instanceof Uint8Array ? (0, bytes_js_1.bytesToUnprefixedHex)(key) : key.toString();
        return this._database.get(dbKey);
    }
    async put(key, val) {
        const dbKey = key instanceof Uint8Array ? (0, bytes_js_1.bytesToUnprefixedHex)(key) : key.toString();
        this._database.set(dbKey, val);
    }
    async del(key) {
        const dbKey = key instanceof Uint8Array ? (0, bytes_js_1.bytesToUnprefixedHex)(key) : key.toString();
        this._database.delete(dbKey);
    }
    async batch(opStack) {
        for (const op of opStack) {
            if (op.type === 'del') {
                await this.del(op.key);
            }
            if (op.type === 'put') {
                await this.put(op.key, op.value);
            }
        }
    }
    /**
     * Note that the returned shallow copy will share the underlying database with the original
     *
     * @returns DB
     */
    shallowCopy() {
        return new MapDB(this._database);
    }
    open() {
        return Promise.resolve();
    }
}
exports.MapDB = MapDB;
//# sourceMappingURL=mapDB.js.map