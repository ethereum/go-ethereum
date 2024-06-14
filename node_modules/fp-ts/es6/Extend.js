import { identity } from './function';
export function duplicate(E) {
    return function (ma) { return E.extend(ma, identity); };
}
