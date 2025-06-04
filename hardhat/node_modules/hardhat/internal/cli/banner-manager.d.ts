export declare class BannerManager {
    private _bannerConfig;
    private _lastDisplayTime;
    private _lastRequestTime;
    private static _instance;
    private constructor();
    static getInstance(): Promise<BannerManager>;
    private _requestBannerConfig;
    private _isBannerConfig;
    showBanner(timeout?: number): Promise<void>;
}
//# sourceMappingURL=banner-manager.d.ts.map