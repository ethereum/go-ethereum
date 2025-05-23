export declare const keyStoreSchema: {
    type: string;
    required: string[];
    properties: {
        crypto: {
            type: string;
            required: string[];
            properties: {
                cipher: {
                    type: string;
                };
                ciphertext: {
                    type: string;
                };
                cipherparams: {
                    type: string;
                };
                kdf: {
                    type: string;
                };
                kdfparams: {
                    type: string;
                };
                salt: {
                    type: string;
                };
                mac: {
                    type: string;
                };
            };
        };
        id: {
            type: string;
        };
        version: {
            type: string;
        };
        address: {
            type: string;
        };
    };
};
//# sourceMappingURL=schemas.d.ts.map