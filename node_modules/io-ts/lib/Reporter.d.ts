import { Validation } from './index';
/**
 * @since 1.0.0
 */
export interface Reporter<A> {
    report: (validation: Validation<any>) => A;
}
