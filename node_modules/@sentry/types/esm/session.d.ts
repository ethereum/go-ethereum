import { User } from './user';
/**
 * @inheritdoc
 */
export interface Session extends SessionContext {
    /** JSDoc */
    update(context?: SessionContext): void;
    /** JSDoc */
    close(status?: SessionStatus): void;
    /** JSDoc */
    toJSON(): {
        init: boolean;
        sid: string;
        did?: string;
        timestamp: string;
        started: string;
        duration: number;
        status: SessionStatus;
        errors: number;
        attrs?: {
            release?: string;
            environment?: string;
            user_agent?: string;
            ip_address?: string;
        };
    };
}
/**
 * Session Context
 */
export interface SessionContext {
    sid?: string;
    did?: string;
    timestamp?: number;
    started?: number;
    duration?: number;
    status?: SessionStatus;
    release?: string;
    environment?: string;
    userAgent?: string;
    ipAddress?: string;
    errors?: number;
    user?: User | null;
}
/**
 * Session Status
 */
export declare enum SessionStatus {
    /** JSDoc */
    Ok = "ok",
    /** JSDoc */
    Exited = "exited",
    /** JSDoc */
    Crashed = "crashed",
    /** JSDoc */
    Abnormal = "abnormal"
}
//# sourceMappingURL=session.d.ts.map