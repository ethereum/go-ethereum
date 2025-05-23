export default function setupBindings(solJson: any): {
    methodFlags: {
        licenseSupported: boolean;
        versionSupported: boolean;
        allocSupported: boolean;
        resetSupported: boolean;
        compileJsonSupported: boolean;
        compileJsonMultiSupported: boolean;
        compileJsonCallbackSuppported: boolean;
        compileJsonStandardSupported: boolean;
    };
    coreBindings: {
        isVersion6OrNewer: boolean;
        addFunction: any;
        removeFunction: any;
        copyFromCString: any;
        copyToCString: any;
        versionToSemver: any;
        alloc: any;
        license: any;
        version: any;
        reset: any;
    };
    compileBindings: {
        compileJson: any;
        compileJsonCallback: (input: any, optimize: any, readCallback: any) => any;
        compileJsonMulti: any;
        compileStandard: any;
    };
};
