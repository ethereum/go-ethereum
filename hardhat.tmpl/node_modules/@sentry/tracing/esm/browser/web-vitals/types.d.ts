export interface Metric {
    name: 'CLS' | 'FCP' | 'FID' | 'LCP' | 'TTFB';
    value: number;
    delta: number;
    id: string;
    isFinal: boolean;
    entries: PerformanceEntry[];
}
export interface ReportHandler {
    (metric: Metric): void;
}
export interface NavigatorNetworkInformation {
    readonly connection?: NetworkInformation;
}
declare type ConnectionType = 'bluetooth' | 'cellular' | 'ethernet' | 'mixed' | 'none' | 'other' | 'unknown' | 'wifi' | 'wimax';
declare type EffectiveConnectionType = '2g' | '3g' | '4g' | 'slow-2g';
declare type Megabit = number;
declare type Millisecond = number;
interface NetworkInformation extends EventTarget {
    readonly type?: ConnectionType;
    readonly effectiveType?: EffectiveConnectionType;
    readonly downlinkMax?: Megabit;
    readonly downlink?: Megabit;
    readonly rtt?: Millisecond;
    readonly saveData?: boolean;
    onchange?: EventListener;
}
export interface NavigatorDeviceMemory {
    readonly deviceMemory?: number;
}
export declare type NavigationTimingPolyfillEntry = Omit<PerformanceNavigationTiming, 'initiatorType' | 'nextHopProtocol' | 'redirectCount' | 'transferSize' | 'encodedBodySize' | 'decodedBodySize' | 'toJSON'>;
export {};
//# sourceMappingURL=types.d.ts.map