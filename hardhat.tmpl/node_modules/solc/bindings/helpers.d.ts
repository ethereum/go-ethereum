export declare function bindSolcMethod(solJson: any, method: any, returnType: any, args: any, defaultValue: any): any;
export declare function bindSolcMethodWithFallbackFunc(solJson: any, method: any, returnType: any, args: any, fallbackMethod: any, finalFallback?: any): any;
export declare function getSupportedMethods(solJson: any): {
    licenseSupported: boolean;
    versionSupported: boolean;
    allocSupported: boolean;
    resetSupported: boolean;
    compileJsonSupported: boolean;
    compileJsonMultiSupported: boolean;
    compileJsonCallbackSuppported: boolean;
    compileJsonStandardSupported: boolean;
};
