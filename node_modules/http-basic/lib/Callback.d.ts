import Response = require('http-response-object');
declare type Callback = (err: NodeJS.ErrnoException | null, response?: Response<NodeJS.ReadableStream>) => void;
export { Callback };
