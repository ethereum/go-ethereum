import { DsnComponents, DsnLike, DsnProtocol } from '@sentry/types';
/** The Sentry Dsn, identifying a Sentry instance and project. */
export declare class Dsn implements DsnComponents {
    /** Protocol used to connect to Sentry. */
    protocol: DsnProtocol;
    /** Public authorization key. */
    user: string;
    /** Private authorization key (deprecated, optional). */
    pass: string;
    /** Hostname of the Sentry instance. */
    host: string;
    /** Port of the Sentry instance. */
    port: string;
    /** Path */
    path: string;
    /** Project ID */
    projectId: string;
    /** Creates a new Dsn component */
    constructor(from: DsnLike);
    /**
     * Renders the string representation of this Dsn.
     *
     * By default, this will render the public representation without the password
     * component. To get the deprecated private representation, set `withPassword`
     * to true.
     *
     * @param withPassword When set to true, the password will be included.
     */
    toString(withPassword?: boolean): string;
    /** Parses a string into this Dsn. */
    private _fromString;
    /** Maps Dsn components into this instance. */
    private _fromComponents;
    /** Validates this Dsn and throws on error. */
    private _validate;
}
//# sourceMappingURL=dsn.d.ts.map