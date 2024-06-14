/// <reference types="node" />
import { HttpVerb } from 'http-basic/lib/HttpVerb';
import { IncomingHttpHeaders } from 'http';
import GenericResponse = require('http-response-object');
import { Options } from './Options';
import { ResponsePromise } from './ResponsePromise';
import { RequestFn } from './RequestFn';
declare type Response = GenericResponse<Buffer | string>;
export { HttpVerb, IncomingHttpHeaders as Headers, Options, ResponsePromise, Response };
declare const fd: any;
export { fd as FormData };
declare const _default: RequestFn;
export default _default;
