declare module "http" {
    import * as events from "events";
    import * as net from "net";
    import * as stream from "stream";
    import { URL } from "url";

    // incoming headers will never contain number
    interface IncomingHttpHeaders {
        'accept-patch'?: string;
        'accept-ranges'?: string;
        'accept'?: string;
        'access-control-allow-credentials'?: string;
        'access-control-allow-headers'?: string;
        'access-control-allow-methods'?: string;
        'access-control-allow-origin'?: string;
        'access-control-expose-headers'?: string;
        'access-control-max-age'?: string;
        'access-control-request-headers'?: string;
        'access-control-request-method'?: string;
        'age'?: string;
        'allow'?: string;
        'alt-svc'?: string;
        'authorization'?: string;
        'cache-control'?: string;
        'connection'?: string;
        'content-disposition'?: string;
        'content-encoding'?: string;
        'content-language'?: string;
        'content-length'?: string;
        'content-location'?: string;
        'content-range'?: string;
        'content-type'?: string;
        'cookie'?: string;
        'date'?: string;
        'etag'?: string;
        'expect'?: string;
        'expires'?: string;
        'forwarded'?: string;
        'from'?: string;
        'host'?: string;
        'if-match'?: string;
        'if-modified-since'?: string;
        'if-none-match'?: string;
        'if-unmodified-since'?: string;
        'last-modified'?: string;
        'location'?: string;
        'origin'?: string;
        'pragma'?: string;
        'proxy-authenticate'?: string;
        'proxy-authorization'?: string;
        'public-key-pins'?: string;
        'range'?: string;
        'referer'?: string;
        'retry-after'?: string;
        'set-cookie'?: string[];
        'strict-transport-security'?: string;
        'tk'?: string;
        'trailer'?: string;
        'transfer-encoding'?: string;
        'upgrade'?: string;
        'user-agent'?: string;
        'vary'?: string;
        'via'?: string;
        'warning'?: string;
        'www-authenticate'?: string;
        [header: string]: string | string[] | undefined;
    }

    // outgoing headers allows numbers (as they are converted internally to strings)
    interface OutgoingHttpHeaders {
        [header: string]: number | string | string[] | undefined;
    }

    interface ClientRequestArgs {
        protocol?: string;
        host?: string;
        hostname?: string;
        family?: number;
        port?: number | string;
        defaultPort?: number | string;
        localAddress?: string;
        socketPath?: string;
        method?: string;
        path?: string;
        headers?: OutgoingHttpHeaders;
        auth?: string;
        agent?: Agent | boolean;
        _defaultAgent?: Agent;
        timeout?: number;
        setHost?: boolean;
        // https://github.com/nodejs/node/blob/master/lib/_http_client.js#L278
        createConnection?: (options: ClientRequestArgs, oncreate: (err: Error, socket: net.Socket) => void) => net.Socket;
    }

    interface ServerOptions {
        IncomingMessage?: typeof IncomingMessage;
        ServerResponse?: typeof ServerResponse;
    }

    type RequestListener = (req: IncomingMessage, res: ServerResponse) => void;

    class Server extends net.Server {
        constructor(requestListener?: RequestListener);
        constructor(options: ServerOptions, requestListener?: RequestListener);

        setTimeout(msecs?: number, callback?: () => void): this;
        setTimeout(callback: () => void): this;
        /**
         * Limits maximum incoming headers count. If set to 0, no limit will be applied.
         * @default 2000
         * {@link https://nodejs.org/api/http.html#http_server_maxheaderscount}
         */
        maxHeadersCount: number | null;
        timeout: number;
        /**
         * Limit the amount of time the parser will wait to receive the complete HTTP headers.
         * @default 40000
         * {@link https://nodejs.org/api/http.html#http_server_headerstimeout}
         */
        headersTimeout: number;
        keepAliveTimeout: number;
    }

    // https://github.com/nodejs/node/blob/master/lib/_http_outgoing.js
    class OutgoingMessage extends stream.Writable {
        upgrading: boolean;
        chunkedEncoding: boolean;
        shouldKeepAlive: boolean;
        useChunkedEncodingByDefault: boolean;
        sendDate: boolean;
        finished: boolean;
        headersSent: boolean;
        connection: net.Socket;

        constructor();

        setTimeout(msecs: number, callback?: () => void): this;
        setHeader(name: string, value: number | string | ReadonlyArray<string>): void;
        getHeader(name: string): number | string | string[] | undefined;
        getHeaders(): OutgoingHttpHeaders;
        getHeaderNames(): string[];
        hasHeader(name: string): boolean;
        removeHeader(name: string): void;
        addTrailers(headers: OutgoingHttpHeaders | ReadonlyArray<[string, string]>): void;
        flushHeaders(): void;
    }

    // https://github.com/nodejs/node/blob/master/lib/_http_server.js#L108-L256
    class ServerResponse extends OutgoingMessage {
        statusCode: number;
        statusMessage: string;

        constructor(req: IncomingMessage);

        assignSocket(socket: net.Socket): void;
        detachSocket(socket: net.Socket): void;
        // https://github.com/nodejs/node/blob/master/test/parallel/test-http-write-callbacks.js#L53
        // no args in writeContinue callback
        writeContinue(callback?: () => void): void;
        writeHead(statusCode: number, reasonPhrase?: string, headers?: OutgoingHttpHeaders): void;
        writeHead(statusCode: number, headers?: OutgoingHttpHeaders): void;
    }

    // https://github.com/nodejs/node/blob/v10.23.0/lib/_http_client.js#L65
    class ClientRequest extends OutgoingMessage {
        connection: net.Socket;
        socket: net.Socket;
        aborted: number;

        constructor(url: string | URL | ClientRequestArgs, cb?: (res: IncomingMessage) => void);

        method: string;
        path: string;
        abort(): void;
        onSocket(socket: net.Socket): void;
        setTimeout(timeout: number, callback?: () => void): this;
        setNoDelay(noDelay?: boolean): void;
        setSocketKeepAlive(enable?: boolean, initialDelay?: number): void;
    }

    class IncomingMessage extends stream.Readable {
        constructor(socket: net.Socket);

        aborted: boolean;
        httpVersion: string;
        httpVersionMajor: number;
        httpVersionMinor: number;
        connection: net.Socket;
        headers: IncomingHttpHeaders;
        rawHeaders: string[];
        trailers: { [key: string]: string | undefined };
        rawTrailers: string[];
        setTimeout(msecs: number, callback?: () => void): this;
        /**
         * Only valid for request obtained from http.Server.
         */
        method?: string;
        /**
         * Only valid for request obtained from http.Server.
         */
        url?: string;
        /**
         * Only valid for response obtained from http.ClientRequest.
         */
        statusCode?: number;
        /**
         * Only valid for response obtained from http.ClientRequest.
         */
        statusMessage?: string;
        socket: net.Socket;
        destroy(error?: Error): void;
    }

    interface AgentOptions {
        /**
         * Keep sockets around in a pool to be used by other requests in the future. Default = false
         */
        keepAlive?: boolean;
        /**
         * When using HTTP KeepAlive, how often to send TCP KeepAlive packets over sockets being kept alive. Default = 1000.
         * Only relevant if keepAlive is set to true.
         */
        keepAliveMsecs?: number;
        /**
         * Maximum number of sockets to allow per host. Default for Node 0.10 is 5, default for Node 0.12 is Infinity
         */
        maxSockets?: number;
        /**
         * Maximum number of sockets to leave open in a free state. Only relevant if keepAlive is set to true. Default = 256.
         */
        maxFreeSockets?: number;
        /**
         * Socket timeout in milliseconds. This will set the timeout after the socket is connected.
         */
        timeout?: number;
    }

    class Agent {
        maxFreeSockets: number;
        maxSockets: number;
        sockets: any;
        requests: any;

        constructor(opts?: AgentOptions);

        /**
         * Destroy any sockets that are currently in use by the agent.
         * It is usually not necessary to do this. However, if you are using an agent with KeepAlive enabled,
         * then it is best to explicitly shut down the agent when you know that it will no longer be used. Otherwise,
         * sockets may hang open for quite a long time before the server terminates them.
         */
        destroy(): void;
    }

    const METHODS: string[];

    const STATUS_CODES: {
        [errorCode: number]: string | undefined;
        [errorCode: string]: string | undefined;
    };

    function createServer(requestListener?: RequestListener): Server;
    function createServer(options: ServerOptions, requestListener?: RequestListener): Server;
    function createClient(port?: number, host?: string): any;

    // although RequestOptions are passed as ClientRequestArgs to ClientRequest directly,
    // create interface RequestOptions would make the naming more clear to developers
    interface RequestOptions extends ClientRequestArgs { }
    function request(options: RequestOptions | string | URL, callback?: (res: IncomingMessage) => void): ClientRequest;
    function request(url: string | URL, options: RequestOptions, callback?: (res: IncomingMessage) => void): ClientRequest;
    function get(options: RequestOptions | string | URL, callback?: (res: IncomingMessage) => void): ClientRequest;
    function get(url: string | URL, options: RequestOptions, callback?: (res: IncomingMessage) => void): ClientRequest;
    let globalAgent: Agent;
}
