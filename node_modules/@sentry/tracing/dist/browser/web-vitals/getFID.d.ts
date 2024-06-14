import { ReportHandler } from './types';
interface FIDPolyfillCallback {
    (value: number, event: Event): void;
}
interface FIDPolyfill {
    onFirstInputDelay: (onReport: FIDPolyfillCallback) => void;
}
declare global {
    interface Window {
        perfMetrics: FIDPolyfill;
    }
}
export declare const getFID: (onReport: ReportHandler) => void;
export {};
//# sourceMappingURL=getFID.d.ts.map