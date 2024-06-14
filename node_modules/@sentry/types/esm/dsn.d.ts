/** Supported Sentry transport protocols in a Dsn. */
export declare type DsnProtocol = 'http' | 'https';
/** Primitive components of a Dsn. */
export interface DsnComponents {
    /** Protocol used to connect to Sentry. */
    protocol: DsnProtocol;
    /** Public authorization key. */
    user: string;
    /** Private authorization key (deprecated, optional). */
    pass?: string;
    /** Hostname of the Sentry instance. */
    host: string;
    /** Port of the Sentry instance. */
    port?: string;
    /** Sub path/ */
    path?: string;
    /** Project ID */
    projectId: string;
}
/** Anything that can be parsed into a Dsn. */
export declare type DsnLike = string | DsnComponents;
/** The Sentry Dsn, identifying a Sentry instance and project. */
export interface Dsn extends DsnComponents {
    /**
     * Renders the string representation of this Dsn.
     *
     * By default, this will render the public representation without the password
     * component. To get the deprecated private representation, set `withPassword`
     * to true.
     *
     * @param withPassword When set to true, the password will be included.
     */
    toString(withPassword: boolean): string;
}
//# sourceMappingURL=dsn.d.ts.map