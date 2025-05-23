import { EventEmitter } from "events";
import {
  EthereumProvider,
  JsonRpcRequest,
  JsonRpcResponse,
  RequestArguments,
} from "../../../types";
import { HardhatError } from "../errors";
import { ERRORS } from "../errors-list";

export type ProviderFactory = () => Promise<EthereumProvider>;
export type Listener = (...args: any[]) => void;

/**
 * A class that delays the (async) creation of its internal provider until the first call
 * to a JSON RPC method via request/send/sendAsync or the init method is called.
 */
export class LazyInitializationProviderAdapter implements EthereumProvider {
  protected provider: EthereumProvider | undefined;
  private _emitter: EventEmitter = new EventEmitter();
  private _initializingPromise: Promise<EthereumProvider> | undefined;

  constructor(private _providerFactory: ProviderFactory) {}

  /**
   * Gets the internal wrapped provider.
   * Using it directly is discouraged and should be done with care,
   * use the public methods from the class like `request` and all event emitter methods instead
   */
  public get _wrapped(): EventEmitter {
    if (this.provider === undefined) {
      throw new HardhatError(ERRORS.GENERAL.UNINITIALIZED_PROVIDER);
    }
    return this.provider;
  }

  public async init(): Promise<EthereumProvider> {
    if (this.provider === undefined) {
      if (this._initializingPromise === undefined) {
        this._initializingPromise = this._providerFactory();
      }
      this.provider = await this._initializingPromise;

      // Copy any event emitter events before initialization over to the provider
      const recordedEvents = this._emitter.eventNames();

      this.provider.setMaxListeners(this._emitter.getMaxListeners());

      for (const event of recordedEvents) {
        const listeners = this._emitter.rawListeners(event) as Listener[];
        for (const listener of listeners) {
          this.provider.on(event, listener);
          this._emitter.removeListener(event, listener);
        }
      }
    }
    return this.provider;
  }

  // Provider methods

  public async request(args: RequestArguments): Promise<unknown> {
    const provider = await this._getOrInitProvider();
    return provider.request(args);
  }

  public async send(method: string, params?: any[]): Promise<any> {
    const provider = await this._getOrInitProvider();
    return provider.send(method, params);
  }

  public sendAsync(
    payload: JsonRpcRequest,
    callback: (error: any, response: JsonRpcResponse) => void
  ): void {
    this._getOrInitProvider().then(
      (provider) => {
        provider.sendAsync(payload, callback);
      },
      (e) => {
        callback(e, null as any);
      }
    );
  }

  // EventEmitter methods

  public addListener(event: string | symbol, listener: EventListener): this {
    this._getEmitter().addListener(event, listener);
    return this;
  }

  public on(event: string | symbol, listener: EventListener): this {
    this._getEmitter().on(event, listener);
    return this;
  }

  public once(event: string | symbol, listener: Listener): this {
    this._getEmitter().once(event, listener);
    return this;
  }

  public prependListener(event: string | symbol, listener: Listener): this {
    this._getEmitter().prependListener(event, listener);
    return this;
  }

  public prependOnceListener(event: string | symbol, listener: Listener): this {
    this._getEmitter().prependOnceListener(event, listener);
    return this;
  }

  public removeListener(event: string | symbol, listener: Listener): this {
    this._getEmitter().removeListener(event, listener);
    return this;
  }

  public off(event: string | symbol, listener: Listener): this {
    this._getEmitter().off(event, listener);
    return this;
  }

  public removeAllListeners(event?: string | symbol | undefined): this {
    this._getEmitter().removeAllListeners(event);
    return this;
  }

  public setMaxListeners(n: number): this {
    this._getEmitter().setMaxListeners(n);
    return this;
  }

  public getMaxListeners(): number {
    return this._getEmitter().getMaxListeners();
  }

  // disable ban-types to satisfy the EventEmitter interface
  // eslint-disable-next-line @typescript-eslint/ban-types
  public listeners(event: string | symbol): Function[] {
    return this._getEmitter().listeners(event);
  }

  // disable ban-types to satisfy the EventEmitter interface
  // eslint-disable-next-line @typescript-eslint/ban-types
  public rawListeners(event: string | symbol): Function[] {
    return this._getEmitter().rawListeners(event);
  }

  public emit(event: string | symbol, ...args: any[]): boolean {
    return this._getEmitter().emit(event, ...args);
  }

  public eventNames(): Array<string | symbol> {
    return this._getEmitter().eventNames();
  }

  public listenerCount(type: string | symbol): number {
    return this._getEmitter().listenerCount(type);
  }

  private _getEmitter(): EventEmitter {
    return this.provider === undefined ? this._emitter : this.provider;
  }

  private async _getOrInitProvider(): Promise<EthereumProvider> {
    // This is here to avoid multiple calls to send async stacking and re-creating the provider
    // over and over again. It shouldn't run for request or send
    if (this._initializingPromise !== undefined) {
      await this._initializingPromise;
    }

    if (this.provider === undefined) {
      this.provider = await this.init();
    }

    return this.provider;
  }
}
