import { Breadcrumb } from './breadcrumb';
import { Contexts } from './context';
import { Exception } from './exception';
import { Extras } from './extra';
import { Primitive } from './misc';
import { Request } from './request';
import { CaptureContext } from './scope';
import { SdkInfo } from './sdkinfo';
import { Severity } from './severity';
import { Span } from './span';
import { Stacktrace } from './stacktrace';
import { Measurements } from './transaction';
import { User } from './user';
/** JSDoc */
export interface Event {
    event_id?: string;
    message?: string;
    timestamp?: number;
    start_timestamp?: number;
    level?: Severity;
    platform?: string;
    logger?: string;
    server_name?: string;
    release?: string;
    dist?: string;
    environment?: string;
    sdk?: SdkInfo;
    request?: Request;
    transaction?: string;
    modules?: {
        [key: string]: string;
    };
    fingerprint?: string[];
    exception?: {
        values?: Exception[];
    };
    stacktrace?: Stacktrace;
    breadcrumbs?: Breadcrumb[];
    contexts?: Contexts;
    tags?: {
        [key: string]: Primitive;
    };
    extra?: Extras;
    user?: User;
    type?: EventType;
    spans?: Span[];
    measurements?: Measurements;
}
/** JSDoc */
export declare type EventType = 'transaction';
/** JSDoc */
export interface EventHint {
    event_id?: string;
    captureContext?: CaptureContext;
    syntheticException?: Error | null;
    originalException?: Error | string | null;
    data?: any;
}
//# sourceMappingURL=event.d.ts.map