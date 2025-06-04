declare module "dgram" {
    import { AddressInfo } from "net";
    import * as dns from "dns";
    import * as events from "events";

    interface RemoteInfo {
        address: string;
        family: string;
        port: number;
    }

    interface BindOptions {
        port: number;
        address?: string;
        exclusive?: boolean;
    }

    type SocketType = "udp4" | "udp6";

    interface SocketOptions {
        type: SocketType;
        reuseAddr?: boolean;
        recvBufferSize?: number;
        sendBufferSize?: number;
        lookup?: (hostname: string, options: dns.LookupOneOptions, callback: (err: NodeJS.ErrnoException | null, address: string, family: number) => void) => void;
    }

    function createSocket(type: SocketType, callback?: (msg: Buffer, rinfo: RemoteInfo) => void): Socket;
    function createSocket(options: SocketOptions, callback?: (msg: Buffer, rinfo: RemoteInfo) => void): Socket;

    class Socket extends events.EventEmitter {
        send(msg: Buffer | string | Uint8Array | ReadonlyArray<any>, port: number, address?: string, callback?: (error: Error | null, bytes: number) => void): void;
        send(msg: Buffer | string | Uint8Array, offset: number, length: number, port: number, address?: string, callback?: (error: Error | null, bytes: number) => void): void;
        bind(port?: number, address?: string, callback?: () => void): void;
        bind(port?: number, callback?: () => void): void;
        bind(callback?: () => void): void;
        bind(options: BindOptions, callback?: Function): void;
        close(callback?: () => void): void;
        address(): AddressInfo | string;
        setBroadcast(flag: boolean): void;
        setTTL(ttl: number): void;
        setMulticastTTL(ttl: number): void;
        setMulticastInterface(multicastInterface: string): void;
        setMulticastLoopback(flag: boolean): void;
        addMembership(multicastAddress: string, multicastInterface?: string): void;
        dropMembership(multicastAddress: string, multicastInterface?: string): void;
        ref(): this;
        unref(): this;
        setRecvBufferSize(size: number): void;
        setSendBufferSize(size: number): void;
        getRecvBufferSize(): number;
        getSendBufferSize(): number;

        /**
         * events.EventEmitter
         * 1. close
         * 2. error
         * 3. listening
         * 4. message
         */
        addListener(event: string, listener: (...args: any[]) => void): this;
        addListener(event: "close", listener: () => void): this;
        addListener(event: "error", listener: (err: Error) => void): this;
        addListener(event: "listening", listener: () => void): this;
        addListener(event: "message", listener: (msg: Buffer, rinfo: AddressInfo) => void): this;

        emit(event: string | symbol, ...args: any[]): boolean;
        emit(event: "close"): boolean;
        emit(event: "error", err: Error): boolean;
        emit(event: "listening"): boolean;
        emit(event: "message", msg: Buffer, rinfo: AddressInfo): boolean;

        on(event: string, listener: (...args: any[]) => void): this;
        on(event: "close", listener: () => void): this;
        on(event: "error", listener: (err: Error) => void): this;
        on(event: "listening", listener: () => void): this;
        on(event: "message", listener: (msg: Buffer, rinfo: AddressInfo) => void): this;

        once(event: string, listener: (...args: any[]) => void): this;
        once(event: "close", listener: () => void): this;
        once(event: "error", listener: (err: Error) => void): this;
        once(event: "listening", listener: () => void): this;
        once(event: "message", listener: (msg: Buffer, rinfo: AddressInfo) => void): this;

        prependListener(event: string, listener: (...args: any[]) => void): this;
        prependListener(event: "close", listener: () => void): this;
        prependListener(event: "error", listener: (err: Error) => void): this;
        prependListener(event: "listening", listener: () => void): this;
        prependListener(event: "message", listener: (msg: Buffer, rinfo: AddressInfo) => void): this;

        prependOnceListener(event: string, listener: (...args: any[]) => void): this;
        prependOnceListener(event: "close", listener: () => void): this;
        prependOnceListener(event: "error", listener: (err: Error) => void): this;
        prependOnceListener(event: "listening", listener: () => void): this;
        prependOnceListener(event: "message", listener: (msg: Buffer, rinfo: AddressInfo) => void): this;
    }
}
