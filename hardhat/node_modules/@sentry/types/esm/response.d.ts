import { Event, EventType } from './event';
import { Session } from './session';
import { Status } from './status';
/** JSDoc */
export interface Response {
    status: Status;
    event?: Event | Session;
    type?: EventType;
    reason?: string;
}
//# sourceMappingURL=response.d.ts.map