/**
 * Ported to Typescript from original implementation below:
 * https://github.com/ahultgren/async-eventemitter -- MIT licensed
 *
 * Type Definitions based on work by: patarapolw <https://github.com/patarapolw> -- MIT licensed
 * that was contributed to Definitely Typed below:
 * https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/async-eventemitter
 */

import { EventEmitter } from 'events'
type AsyncListener<T, R> =
  | ((data: T, callback?: (result?: R) => void) => Promise<R>)
  | ((data: T, callback?: (result?: R) => void) => void)
export interface EventMap {
  [event: string]: AsyncListener<any, any>
}

async function runInSeries(
  context: any,
  tasks: Array<(data: unknown, callback?: (error?: Error) => void) => void>,
  data: unknown
): Promise<void> {
  let error: Error | undefined
  for await (const task of tasks) {
    try {
      if (task.length < 2) {
        //sync
        task.call(context, data)
      } else {
        await new Promise<void>((resolve, reject) => {
          task.call(context, data, (error) => {
            if (error) {
              reject(error)
            } else {
              resolve()
            }
          })
        })
      }
    } catch (e: unknown) {
      error = e as Error
    }
  }
  if (error) {
    throw error
  }
}

export class AsyncEventEmitter<T extends EventMap> extends EventEmitter {
  emit<E extends keyof T>(event: E & string, ...args: Parameters<T[E]>) {
    let [data, callback] = args
    const self = this

    let listeners = (self as any)._events[event] ?? []

    // Optional data argument
    if (callback === undefined && typeof data === 'function') {
      callback = data
      data = undefined
    }

    // Special treatment of internal newListener and removeListener events
    if (event === 'newListener' || event === 'removeListener') {
      data = {
        event: data,
        fn: callback,
      }

      callback = undefined
    }

    // A single listener is just a function not an array...
    listeners = Array.isArray(listeners) ? listeners : [listeners]
    runInSeries(self, listeners.slice(), data).then(callback).catch(callback)

    return self.listenerCount(event) > 0
  }

  once<E extends keyof T>(event: E & string, listener: T[E]): this {
    const self = this
    let g: (...args: any[]) => void

    if (typeof listener !== 'function') {
      throw new TypeError('listener must be a function')
    }

    // Hack to support set arity
    if (listener.length >= 2) {
      g = function (e: E, next: any) {
        self.removeListener(event, g as T[E])
        void listener(e, next)
      }
    } else {
      g = function (e: E) {
        self.removeListener(event, g as T[E])
        void listener(e, g)
      }
    }

    self.on(event, g as T[E])

    return self
  }

  first<E extends keyof T>(event: E & string, listener: T[E]): this {
    let listeners = (this as any)._events[event] ?? []

    // Contract
    if (typeof listener !== 'function') {
      throw new TypeError('listener must be a function')
    }

    // Listeners are not always an array
    if (!Array.isArray(listeners)) {
      ;(this as any)._events[event] = listeners = [listeners]
    }

    listeners.unshift(listener)

    return this
  }

  before<E extends keyof T>(event: E & string, target: T[E], listener: T[E]): this {
    return this.beforeOrAfter(event, target, listener)
  }

  after<E extends keyof T>(event: E & string, target: T[E], listener: T[E]): this {
    return this.beforeOrAfter(event, target, listener, 'after')
  }

  private beforeOrAfter<E extends keyof T>(
    event: E & string,
    target: T[E],
    listener: T[E],
    beforeOrAfter?: string
  ) {
    let listeners = (this as any)._events[event] ?? []
    let i
    let index
    const add = beforeOrAfter === 'after' ? 1 : 0

    // Contract
    if (typeof listener !== 'function') {
      throw new TypeError('listener must be a function')
    }
    if (typeof target !== 'function') {
      throw new TypeError('target must be a function')
    }

    // Listeners are not always an array
    if (!Array.isArray(listeners)) {
      ;(this as any)._events[event] = listeners = [listeners]
    }

    index = listeners.length

    for (i = listeners.length; i--; ) {
      if (listeners[i] === target) {
        index = i + add
        break
      }
    }

    listeners.splice(index, 0, listener)

    return this
  }

  on<E extends keyof T>(event: E & string, listener: T[E]): this {
    return super.on(event, listener)
  }

  addListener<E extends keyof T>(event: E & string, listener: T[E]): this {
    return super.addListener(event, listener)
  }

  prependListener<E extends keyof T>(event: E & string, listener: T[E]): this {
    return super.prependListener(event, listener)
  }

  prependOnceListener<E extends keyof T>(event: E & string, listener: T[E]): this {
    return super.prependOnceListener(event, listener)
  }

  removeAllListeners(event?: keyof T & string): this {
    return super.removeAllListeners(event)
  }

  removeListener<E extends keyof T>(event: E & string, listener: T[E]): this {
    return super.removeListener(event, listener)
  }

  eventNames(): Array<keyof T & string> {
    return super.eventNames() as keyof T & string[]
  }

  listeners<E extends keyof T>(event: E & string): Array<T[E]> {
    return super.listeners(event) as T[E][]
  }

  listenerCount(event: keyof T & string): number {
    return super.listenerCount(event)
  }

  getMaxListeners(): number {
    return super.getMaxListeners()
  }

  setMaxListeners(maxListeners: number): this {
    return super.setMaxListeners(maxListeners)
  }
}
