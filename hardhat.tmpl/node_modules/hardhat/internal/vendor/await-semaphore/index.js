"use strict";
// Based on: https://github.com/notenoughneon/await-semaphore/blob/f117a6b59324038c9e8ee04c70c328215a727812/index.ts
// which is distributed under this license: https://github.com/notenoughneon/await-semaphore/blob/f117a6b59324038c9e8ee04c70c328215a727812/LICENSE
Object.defineProperty(exports, "__esModule", { value: true });
exports.Mutex = exports.Semaphore = void 0;
/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */
class Semaphore {
    constructor(count) {
        this._tasks = [];
        this.count = count;
    }
    acquire() {
        return new Promise((res) => {
            const task = () => {
                let released = false;
                res(() => {
                    if (!released) {
                        released = true;
                        this.count++;
                        this._sched();
                    }
                });
            };
            this._tasks.push(task);
            if (process !== undefined && process.nextTick !== undefined) {
                process.nextTick(this._sched.bind(this));
            }
            else {
                setImmediate(this._sched.bind(this));
            }
        });
    }
    use(f) {
        return this.acquire().then((release) => {
            return f()
                .then((res) => {
                release();
                return res;
            })
                .catch((err) => {
                release();
                throw err;
            });
        });
    }
    _sched() {
        if (this.count > 0 && this._tasks.length > 0) {
            this.count--;
            const next = this._tasks.shift();
            if (next === undefined) {
                throw new Error("Unexpected undefined value in tasks list");
            }
            next();
        }
    }
}
exports.Semaphore = Semaphore;
class Mutex extends Semaphore {
    constructor() {
        super(1);
    }
}
exports.Mutex = Mutex;
//# sourceMappingURL=index.js.map