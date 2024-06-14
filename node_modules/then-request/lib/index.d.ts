/// <reference types="node" />
import GenericResponse = require('http-response-object');
import { IncomingHttpHeaders } from 'http';
import { Options } from './Options';
import { ResponsePromise } from './ResponsePromise';
import { RequestFn } from './RequestFn';
import { HttpVerb } from 'http-basic';
import FormData = require('form-data');
declare type Response = GenericResponse<Buffer | string>;
export { HttpVerb, IncomingHttpHeaders as Headers, Options, ResponsePromise, Response };
export { FormData };
declare const _default: RequestFn;
export default _default;
