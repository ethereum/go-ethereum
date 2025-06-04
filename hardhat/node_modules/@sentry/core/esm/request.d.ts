import { Event, SentryRequest, Session } from '@sentry/types';
import { API } from './api';
/** Creates a SentryRequest from an event. */
export declare function sessionToSentryRequest(session: Session, api: API): SentryRequest;
/** Creates a SentryRequest from an event. */
export declare function eventToSentryRequest(event: Event, api: API): SentryRequest;
//# sourceMappingURL=request.d.ts.map