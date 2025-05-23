import { EventEmitter } from "events";

// IMPORTANT NOTE: This class is type-checked against the currently installed
// version of @types/node (10.x atm), and manually checked to be compatible with
// Node.js up to 14.3.0 (the latest release atm). There's a test that ensures
// that we are exporting all the EventEmitter's members, but it can't check the
// actual types of those members if they are functions.
//
// If a new version of Node.js adds new members to EventEmitter or overloads
// existing ones this class has to be updated, even if it still type-checks.
// This is a serious limitation ot DefinitelyTyped when the original, un-typed,
// library can change because of the user having a different version.
export class EventEmitterWrapper implements EventEmitter {
  constructor(private readonly _wrapped: EventEmitter) {}

  public addListener(
    event: string | symbol,
    listener: (...args: any[]) => void
  ): this {
    this._wrapped.addListener(event, listener);
    return this;
  }

  public on(event: string | symbol, listener: (...args: any[]) => void): this {
    this._wrapped.on(event, listener);
    return this;
  }

  public once(
    event: string | symbol,
    listener: (...args: any[]) => void
  ): this {
    this._wrapped.once(event, listener);
    return this;
  }

  public prependListener(
    event: string | symbol,
    listener: (...args: any[]) => void
  ): this {
    this._wrapped.prependListener(event, listener);
    return this;
  }

  public prependOnceListener(
    event: string | symbol,
    listener: (...args: any[]) => void
  ): this {
    this._wrapped.prependOnceListener(event, listener);
    return this;
  }

  public removeListener(
    event: string | symbol,
    listener: (...args: any[]) => void
  ): this {
    this._wrapped.removeListener(event, listener);
    return this;
  }

  public off(event: string | symbol, listener: (...args: any[]) => void): this {
    this._wrapped.off(event, listener);
    return this;
  }

  public removeAllListeners(event?: string | symbol | undefined): this {
    this._wrapped.removeAllListeners(event);
    return this;
  }

  public setMaxListeners(n: number): this {
    this._wrapped.setMaxListeners(n);
    return this;
  }

  public getMaxListeners(): number {
    return this._wrapped.getMaxListeners();
  }

  // eslint-disable-next-line @typescript-eslint/ban-types
  public listeners(event: string | symbol): Function[] {
    return this._wrapped.listeners(event);
  }

  // eslint-disable-next-line @typescript-eslint/ban-types
  public rawListeners(event: string | symbol): Function[] {
    return this._wrapped.rawListeners(event);
  }

  public emit(event: string | symbol, ...args: any[]): boolean {
    return this._wrapped.emit(event, ...args);
  }

  public eventNames(): Array<string | symbol> {
    return this._wrapped.eventNames();
  }

  public listenerCount(type: string | symbol): number {
    return this._wrapped.listenerCount(type);
  }
}
