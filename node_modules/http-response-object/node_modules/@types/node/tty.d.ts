declare module "tty" {
    import * as net from "net";

    function isatty(fd: number): boolean;
    class ReadStream extends net.Socket {
        constructor(fd: number, options?: net.SocketConstructorOpts);
        isRaw: boolean;
        setRawMode(mode: boolean): this;
        isTTY: boolean;
    }
    class WriteStream extends net.Socket {
        constructor(fd: number);
        columns: number;
        rows: number;
        isTTY: boolean;
    }
}
