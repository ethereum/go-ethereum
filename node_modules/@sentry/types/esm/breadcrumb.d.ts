import { Severity } from './severity';
/** JSDoc */
export interface Breadcrumb {
    type?: string;
    level?: Severity;
    event_id?: string;
    category?: string;
    message?: string;
    data?: {
        [key: string]: any;
    };
    timestamp?: number;
}
/** JSDoc */
export interface BreadcrumbHint {
    [key: string]: any;
}
//# sourceMappingURL=breadcrumb.d.ts.map