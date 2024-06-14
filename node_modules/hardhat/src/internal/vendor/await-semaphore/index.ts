// Based on: https://github.com/notenoughneon/await-semaphore/blob/f117a6b59324038c9e8ee04c70c328215a727812/index.ts
// which is distributed under this license: https://github.com/notenoughneon/await-semaphore/blob/f117a6b59324038c9e8ee04c70c328215a727812/LICENSE

/* eslint-disable @nomicfoundation/hardhat-internal-rules/only-hardhat-error */

export class Semaphore {
  public count: number;
  private _tasks: Array<() => void> = [];

  constructor(count: number) {
    this.count = count;
  }

  public acquire() {
    return new Promise<() => void>((res) => {
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
      } else {
        setImmediate(this._sched.bind(this));
      }
    });
  }

  public use<T>(f: () => Promise<T>) {
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

  private _sched() {
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

export class Mutex extends Semaphore {
  constructor() {
    super(1);
  }
}
