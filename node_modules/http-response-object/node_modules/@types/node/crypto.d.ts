declare module 'crypto' {
    import * as stream from 'stream';

    interface Certificate {
        exportChallenge(spkac: string | Buffer | NodeJS.TypedArray | DataView): Buffer;
        exportPublicKey(spkac: string | Buffer | NodeJS.TypedArray | DataView): Buffer;
        verifySpkac(spkac: Buffer | NodeJS.TypedArray | DataView): boolean;
    }
    const Certificate: {
        new (): Certificate;
        (): Certificate;
    };

    /** @deprecated since v10.0.0 */
    const fips: boolean;

    interface CredentialDetails {
        pfx: string;
        key: string;
        passphrase: string;
        cert: string;
        ca: string | string[];
        crl: string | string[];
        ciphers: string;
    }
    /** @deprecated since v0.11.13 - use tls.SecureContext instead. */
    interface Credentials {
        context?: any;
    }
    /** @deprecated since v0.11.13 - use tls.createSecureContext instead. */
    function createCredentials(details: CredentialDetails): Credentials;
    function createHash(algorithm: string, options?: stream.TransformOptions): Hash;
    function createHmac(
        algorithm: string,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        options?: stream.TransformOptions,
    ): Hmac;

    type Utf8AsciiLatin1Encoding = 'utf8' | 'ascii' | 'latin1';
    type HexBase64Latin1Encoding = 'latin1' | 'hex' | 'base64';
    type Utf8AsciiBinaryEncoding = 'utf8' | 'ascii' | 'binary';
    type HexBase64BinaryEncoding = 'binary' | 'base64' | 'hex';
    type ECDHKeyFormat = 'compressed' | 'uncompressed' | 'hybrid';

    interface Hash extends stream.Transform {
        update(data: string | Buffer | NodeJS.TypedArray | DataView): Hash;
        update(data: string, input_encoding: Utf8AsciiLatin1Encoding): Hash;
        digest(): Buffer;
        digest(encoding: HexBase64Latin1Encoding): string;
    }
    interface Hmac extends stream.Transform {
        update(data: string | Buffer | NodeJS.TypedArray | DataView): Hmac;
        update(data: string, input_encoding: Utf8AsciiLatin1Encoding): Hmac;
        digest(): Buffer;
        digest(encoding: HexBase64Latin1Encoding): string;
    }
    type CipherCCMTypes = 'aes-128-ccm' | 'aes-192-ccm' | 'aes-256-ccm';
    type CipherGCMTypes = 'aes-128-gcm' | 'aes-192-gcm' | 'aes-256-gcm';
    interface CipherCCMOptions extends stream.TransformOptions {
        authTagLength: number;
    }
    interface CipherGCMOptions extends stream.TransformOptions {
        authTagLength?: number;
    }
    /** @deprecated since v10.0.0 use createCipheriv() */
    function createCipher(
        algorithm: CipherCCMTypes,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options: CipherCCMOptions,
    ): CipherCCM;
    /** @deprecated since v10.0.0 use createCipheriv() */
    function createCipher(
        algorithm: CipherGCMTypes,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options?: CipherGCMOptions,
    ): CipherGCM;
    /** @deprecated since v10.0.0 use createCipheriv() */
    function createCipher(
        algorithm: string,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options?: stream.TransformOptions,
    ): Cipher;

    function createCipheriv(
        algorithm: CipherCCMTypes,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options: CipherCCMOptions,
    ): CipherCCM;
    function createCipheriv(
        algorithm: CipherGCMTypes,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options?: CipherGCMOptions,
    ): CipherGCM;
    function createCipheriv(
        algorithm: string,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options?: stream.TransformOptions,
    ): Cipher;

    interface Cipher extends stream.Transform {
        update(data: string | Buffer | NodeJS.TypedArray | DataView): Buffer;
        update(data: string, input_encoding: Utf8AsciiBinaryEncoding): Buffer;
        update(data: Buffer | NodeJS.TypedArray | DataView, output_encoding: HexBase64BinaryEncoding): string;
        update(
            data: Buffer | NodeJS.TypedArray | DataView,
            input_encoding: any,
            output_encoding: HexBase64BinaryEncoding,
        ): string;
        // second arg ignored
        update(data: string, input_encoding: Utf8AsciiBinaryEncoding, output_encoding: HexBase64BinaryEncoding): string;
        final(): Buffer;
        final(output_encoding: string): string;
        setAutoPadding(auto_padding?: boolean): this;
        // getAuthTag(): Buffer;
        // setAAD(buffer: Buffer): this; // docs only say buffer
    }
    interface CipherCCM extends Cipher {
        setAAD(buffer: Buffer, options: { plaintextLength: number }): this;
        getAuthTag(): Buffer;
    }
    interface CipherGCM extends Cipher {
        setAAD(buffer: Buffer, options?: { plaintextLength: number }): this;
        getAuthTag(): Buffer;
    }
    /** @deprecated since v10.0.0 use createDecipheriv() */
    function createDecipher(
        algorithm: CipherCCMTypes,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options: CipherCCMOptions,
    ): DecipherCCM;
    /** @deprecated since v10.0.0 use createDecipheriv() */
    function createDecipher(
        algorithm: CipherGCMTypes,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options?: CipherGCMOptions,
    ): DecipherGCM;
    /** @deprecated since v10.0.0 use createDecipheriv() */
    function createDecipher(
        algorithm: string,
        password: string | Buffer | NodeJS.TypedArray | DataView,
        options?: stream.TransformOptions,
    ): Decipher;

    function createDecipheriv(
        algorithm: CipherCCMTypes,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options: CipherCCMOptions,
    ): DecipherCCM;
    function createDecipheriv(
        algorithm: CipherGCMTypes,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options?: CipherGCMOptions,
    ): DecipherGCM;
    function createDecipheriv(
        algorithm: string,
        key: string | Buffer | NodeJS.TypedArray | DataView,
        iv: string | Buffer | NodeJS.TypedArray | DataView,
        options?: stream.TransformOptions,
    ): Decipher;

    interface Decipher extends stream.Transform {
        update(data: Buffer | NodeJS.TypedArray | DataView): Buffer;
        update(data: string, input_encoding: HexBase64BinaryEncoding): Buffer;
        update(
            data: Buffer | NodeJS.TypedArray | DataView,
            input_encoding: HexBase64BinaryEncoding | undefined,
            output_encoding: Utf8AsciiBinaryEncoding,
        ): string;
        // second arg is ignored
        update(data: string, input_encoding: HexBase64BinaryEncoding, output_encoding: Utf8AsciiBinaryEncoding): string;
        final(): Buffer;
        final(output_encoding: string): string;
        setAutoPadding(auto_padding?: boolean): this;
        // setAuthTag(tag: Buffer | NodeJS.TypedArray | DataView): this;
        // setAAD(buffer: Buffer | NodeJS.TypedArray | DataView): this;
    }
    interface DecipherCCM extends Decipher {
        setAuthTag(buffer: Buffer | NodeJS.TypedArray | DataView): this;
        setAAD(buffer: Buffer | NodeJS.TypedArray | DataView, options: { plaintextLength: number }): this;
    }
    interface DecipherGCM extends Decipher {
        setAuthTag(buffer: Buffer | NodeJS.TypedArray | DataView): this;
        setAAD(buffer: Buffer | NodeJS.TypedArray | DataView, options?: { plaintextLength: number }): this;
    }

    function createSign(algorithm: string, options?: stream.WritableOptions): Signer;
    interface Signer extends NodeJS.WritableStream {
        update(data: string | Buffer | NodeJS.TypedArray | DataView): Signer;
        update(data: string, input_encoding: Utf8AsciiLatin1Encoding): Signer;
        sign(private_key: string | { key: string; passphrase?: string; padding?: number; saltLength?: number }): Buffer;
        sign(
            private_key: string | { key: string; passphrase?: string; padding?: number; saltLength?: number },
            output_format: HexBase64Latin1Encoding,
        ): string;
    }
    function createVerify(algorith: string, options?: stream.WritableOptions): Verify;
    interface Verify extends NodeJS.WritableStream {
        update(data: string | Buffer | NodeJS.TypedArray | DataView): Verify;
        update(data: string, input_encoding: Utf8AsciiLatin1Encoding): Verify;
        verify(object: string | Object, signature: Buffer | NodeJS.TypedArray | DataView): boolean;
        verify(object: string | Object, signature: string, signature_format: HexBase64Latin1Encoding): boolean;
        // https://nodejs.org/api/crypto.html#crypto_verifier_verify_object_signature_signature_format
        // The signature field accepts a TypedArray type, but it is only available starting ES2017
    }
    function createDiffieHellman(
        prime_length: number,
        generator?: number | Buffer | NodeJS.TypedArray | DataView,
    ): DiffieHellman;
    function createDiffieHellman(prime: Buffer | NodeJS.TypedArray | DataView): DiffieHellman;
    function createDiffieHellman(prime: string, prime_encoding: HexBase64Latin1Encoding): DiffieHellman;
    function createDiffieHellman(
        prime: string,
        prime_encoding: HexBase64Latin1Encoding,
        generator: number | Buffer | NodeJS.TypedArray | DataView,
    ): DiffieHellman;
    function createDiffieHellman(
        prime: string,
        prime_encoding: HexBase64Latin1Encoding,
        generator: string,
        generator_encoding: HexBase64Latin1Encoding,
    ): DiffieHellman;
    interface DiffieHellman {
        generateKeys(): Buffer;
        generateKeys(encoding: HexBase64Latin1Encoding): string;
        computeSecret(other_public_key: Buffer | NodeJS.TypedArray | DataView): Buffer;
        computeSecret(other_public_key: string, input_encoding: HexBase64Latin1Encoding): Buffer;
        computeSecret(
            other_public_key: Buffer | NodeJS.TypedArray | DataView,
            output_encoding: HexBase64Latin1Encoding,
        ): string;
        computeSecret(
            other_public_key: string,
            input_encoding: HexBase64Latin1Encoding,
            output_encoding: HexBase64Latin1Encoding,
        ): string;
        getPrime(): Buffer;
        getPrime(encoding: HexBase64Latin1Encoding): string;
        getGenerator(): Buffer;
        getGenerator(encoding: HexBase64Latin1Encoding): string;
        getPublicKey(): Buffer;
        getPublicKey(encoding: HexBase64Latin1Encoding): string;
        getPrivateKey(): Buffer;
        getPrivateKey(encoding: HexBase64Latin1Encoding): string;
        setPublicKey(public_key: Buffer | NodeJS.TypedArray | DataView): void;
        setPublicKey(public_key: string, encoding: string): void;
        setPrivateKey(private_key: Buffer | NodeJS.TypedArray | DataView): void;
        setPrivateKey(private_key: string, encoding: string): void;
        verifyError: number;
    }
    function getDiffieHellman(group_name: string): DiffieHellman;
    function pbkdf2(
        password: string | Buffer | NodeJS.TypedArray | DataView,
        salt: string | Buffer | NodeJS.TypedArray | DataView,
        iterations: number,
        keylen: number,
        digest: string,
        callback: (err: Error | null, derivedKey: Buffer) => any,
    ): void;
    function pbkdf2Sync(
        password: string | Buffer | NodeJS.TypedArray | DataView,
        salt: string | Buffer | NodeJS.TypedArray | DataView,
        iterations: number,
        keylen: number,
        digest: string,
    ): Buffer;

    function randomBytes(size: number): Buffer;
    function randomBytes(size: number, callback: (err: Error | null, buf: Buffer) => void): void;
    function pseudoRandomBytes(size: number): Buffer;
    function pseudoRandomBytes(size: number, callback: (err: Error | null, buf: Buffer) => void): void;

    function randomFillSync<T extends Buffer | NodeJS.TypedArray | DataView>(
        buffer: T,
        offset?: number,
        size?: number,
    ): T;
    function randomFill<T extends Buffer | NodeJS.TypedArray | DataView>(
        buffer: T,
        callback: (err: Error | null, buf: T) => void,
    ): void;
    function randomFill<T extends Buffer | NodeJS.TypedArray | DataView>(
        buffer: T,
        offset: number,
        callback: (err: Error | null, buf: T) => void,
    ): void;
    function randomFill<T extends Buffer | NodeJS.TypedArray | DataView>(
        buffer: T,
        offset: number,
        size: number,
        callback: (err: Error | null, buf: T) => void,
    ): void;

    interface ScryptOptions {
        cost?: number;
        blockSize?: number;
        parallelization?: number;
        N?: number;
        r?: number;
        p?: number;
        maxmem?: number;
    }
    function scrypt(
        password: string | Buffer | NodeJS.TypedArray | DataView,
        salt: string | Buffer | NodeJS.TypedArray | DataView,
        keylen: number,
        callback: (err: Error | null, derivedKey: Buffer) => void,
    ): void;
    function scrypt(
        password: string | Buffer | NodeJS.TypedArray | DataView,
        salt: string | Buffer | NodeJS.TypedArray | DataView,
        keylen: number,
        options: ScryptOptions,
        callback: (err: Error | null, derivedKey: Buffer) => void,
    ): void;
    function scryptSync(
        password: string | Buffer | NodeJS.TypedArray | DataView,
        salt: string | Buffer | NodeJS.TypedArray | DataView,
        keylen: number,
        options?: ScryptOptions,
    ): Buffer;

    interface RsaPublicKey {
        key: string;
        padding?: number;
    }
    interface RsaPrivateKey {
        key: string;
        passphrase?: string;
        padding?: number;
    }
    function publicEncrypt(public_key: string | RsaPublicKey, buffer: Buffer | NodeJS.TypedArray | DataView): Buffer;
    function privateDecrypt(private_key: string | RsaPrivateKey, buffer: Buffer | NodeJS.TypedArray | DataView): Buffer;
    function privateEncrypt(private_key: string | RsaPrivateKey, buffer: Buffer | NodeJS.TypedArray | DataView): Buffer;
    function publicDecrypt(public_key: string | RsaPublicKey, buffer: Buffer | NodeJS.TypedArray | DataView): Buffer;
    function getCiphers(): string[];
    function getCurves(): string[];
    function getFips(): 1 | 0;
    function getHashes(): string[];
    class ECDH {
        static convertKey(
            key: string | Buffer | NodeJS.TypedArray | DataView,
            curve: string,
            inputEncoding?: HexBase64Latin1Encoding,
            outputEncoding?: 'latin1' | 'hex' | 'base64',
            format?: 'uncompressed' | 'compressed' | 'hybrid',
        ): Buffer | string;
        generateKeys(): Buffer;
        generateKeys(encoding: HexBase64Latin1Encoding, format?: ECDHKeyFormat): string;
        computeSecret(other_public_key: Buffer | NodeJS.TypedArray | DataView): Buffer;
        computeSecret(other_public_key: string, input_encoding: HexBase64Latin1Encoding): Buffer;
        computeSecret(
            other_public_key: Buffer | NodeJS.TypedArray | DataView,
            output_encoding: HexBase64Latin1Encoding,
        ): string;
        computeSecret(
            other_public_key: string,
            input_encoding: HexBase64Latin1Encoding,
            output_encoding: HexBase64Latin1Encoding,
        ): string;
        getPrivateKey(): Buffer;
        getPrivateKey(encoding: HexBase64Latin1Encoding): string;
        getPublicKey(): Buffer;
        getPublicKey(encoding: HexBase64Latin1Encoding, format?: ECDHKeyFormat): string;
        setPrivateKey(private_key: Buffer | NodeJS.TypedArray | DataView): void;
        setPrivateKey(private_key: string, encoding: HexBase64Latin1Encoding): void;
    }
    function createECDH(curve_name: string): ECDH;
    function timingSafeEqual(
        a: Buffer | NodeJS.TypedArray | DataView,
        b: Buffer | NodeJS.TypedArray | DataView,
    ): boolean;
    /** @deprecated since v10.0.0 */
    const DEFAULT_ENCODING: string;

    export type KeyType = 'rsa' | 'dsa' | 'ec';
    export type KeyFormat = 'pem' | 'der';

    interface BasePrivateKeyEncodingOptions<T extends KeyFormat> {
        format: T;
        cipher?: string;
        passphrase?: string;
    }

    interface RSAKeyPairOptions<PubF extends KeyFormat, PrivF extends KeyFormat> {
        /**
         * Key size in bits
         */
        modulusLength: number;
        /**
         * @default 0x10001
         */
        publicExponent?: number;

        publicKeyEncoding: {
            type: 'pkcs1' | 'spki';
            format: PubF;
        };
        privateKeyEncoding: BasePrivateKeyEncodingOptions<PrivF> & {
            type: 'pkcs1' | 'pkcs8';
        };
    }

    interface DSAKeyPairOptions<PubF extends KeyFormat, PrivF extends KeyFormat> {
        /**
         * Key size in bits
         */
        modulusLength: number;
        /**
         * Size of q in bits
         */
        divisorLength: number;

        publicKeyEncoding: {
            type: 'spki';
            format: PubF;
        };
        privateKeyEncoding: BasePrivateKeyEncodingOptions<PrivF> & {
            type: 'pkcs8';
        };
    }

    interface ECKeyPairOptions<PubF extends KeyFormat, PrivF extends KeyFormat> {
        /**
         * Name of the curve to use.
         */
        namedCurve: string;

        publicKeyEncoding: {
            type: 'pkcs1' | 'spki';
            format: PubF;
        };
        privateKeyEncoding: BasePrivateKeyEncodingOptions<PrivF> & {
            type: 'sec1' | 'pkcs8';
        };
    }

    interface KeyPairSyncResult<T1 extends string | Buffer, T2 extends string | Buffer> {
        publicKey: T1;
        privateKey: T2;
    }

    function generateKeyPairSync(
        type: 'rsa',
        options: RSAKeyPairOptions<'pem', 'pem'>,
    ): KeyPairSyncResult<string, string>;
    function generateKeyPairSync(
        type: 'rsa',
        options: RSAKeyPairOptions<'pem', 'der'>,
    ): KeyPairSyncResult<string, Buffer>;
    function generateKeyPairSync(
        type: 'rsa',
        options: RSAKeyPairOptions<'der', 'pem'>,
    ): KeyPairSyncResult<Buffer, string>;
    function generateKeyPairSync(
        type: 'rsa',
        options: RSAKeyPairOptions<'der', 'der'>,
    ): KeyPairSyncResult<Buffer, Buffer>;

    function generateKeyPairSync(
        type: 'dsa',
        options: DSAKeyPairOptions<'pem', 'pem'>,
    ): KeyPairSyncResult<string, string>;
    function generateKeyPairSync(
        type: 'dsa',
        options: DSAKeyPairOptions<'pem', 'der'>,
    ): KeyPairSyncResult<string, Buffer>;
    function generateKeyPairSync(
        type: 'dsa',
        options: DSAKeyPairOptions<'der', 'pem'>,
    ): KeyPairSyncResult<Buffer, string>;
    function generateKeyPairSync(
        type: 'dsa',
        options: DSAKeyPairOptions<'der', 'der'>,
    ): KeyPairSyncResult<Buffer, Buffer>;

    function generateKeyPairSync(
        type: 'ec',
        options: ECKeyPairOptions<'pem', 'pem'>,
    ): KeyPairSyncResult<string, string>;
    function generateKeyPairSync(
        type: 'ec',
        options: ECKeyPairOptions<'pem', 'der'>,
    ): KeyPairSyncResult<string, Buffer>;
    function generateKeyPairSync(
        type: 'ec',
        options: ECKeyPairOptions<'der', 'pem'>,
    ): KeyPairSyncResult<Buffer, string>;
    function generateKeyPairSync(
        type: 'ec',
        options: ECKeyPairOptions<'der', 'der'>,
    ): KeyPairSyncResult<Buffer, Buffer>;

    function generateKeyPair(
        type: 'rsa',
        options: RSAKeyPairOptions<'pem', 'pem'>,
        callback: (err: Error | null, publicKey: string, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'rsa',
        options: RSAKeyPairOptions<'pem', 'der'>,
        callback: (err: Error | null, publicKey: string, privateKey: Buffer) => void,
    ): void;
    function generateKeyPair(
        type: 'rsa',
        options: RSAKeyPairOptions<'der', 'pem'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'rsa',
        options: RSAKeyPairOptions<'der', 'der'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: Buffer) => void,
    ): void;

    function generateKeyPair(
        type: 'dsa',
        options: DSAKeyPairOptions<'pem', 'pem'>,
        callback: (err: Error | null, publicKey: string, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'dsa',
        options: DSAKeyPairOptions<'pem', 'der'>,
        callback: (err: Error | null, publicKey: string, privateKey: Buffer) => void,
    ): void;
    function generateKeyPair(
        type: 'dsa',
        options: DSAKeyPairOptions<'der', 'pem'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'dsa',
        options: DSAKeyPairOptions<'der', 'der'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: Buffer) => void,
    ): void;

    function generateKeyPair(
        type: 'ec',
        options: ECKeyPairOptions<'pem', 'pem'>,
        callback: (err: Error | null, publicKey: string, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'ec',
        options: ECKeyPairOptions<'pem', 'der'>,
        callback: (err: Error | null, publicKey: string, privateKey: Buffer) => void,
    ): void;
    function generateKeyPair(
        type: 'ec',
        options: ECKeyPairOptions<'der', 'pem'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: string) => void,
    ): void;
    function generateKeyPair(
        type: 'ec',
        options: ECKeyPairOptions<'der', 'der'>,
        callback: (err: Error | null, publicKey: Buffer, privateKey: Buffer) => void,
    ): void;

    namespace generateKeyPair {
        function __promisify__(
            type: 'rsa',
            options: RSAKeyPairOptions<'pem', 'pem'>,
        ): Promise<{ publicKey: string; privateKey: string }>;
        function __promisify__(
            type: 'rsa',
            options: RSAKeyPairOptions<'pem', 'der'>,
        ): Promise<{ publicKey: string; privateKey: Buffer }>;
        function __promisify__(
            type: 'rsa',
            options: RSAKeyPairOptions<'der', 'pem'>,
        ): Promise<{ publicKey: Buffer; privateKey: string }>;
        function __promisify__(
            type: 'rsa',
            options: RSAKeyPairOptions<'der', 'der'>,
        ): Promise<{ publicKey: Buffer; privateKey: Buffer }>;

        function __promisify__(
            type: 'dsa',
            options: DSAKeyPairOptions<'pem', 'pem'>,
        ): Promise<{ publicKey: string; privateKey: string }>;
        function __promisify__(
            type: 'dsa',
            options: DSAKeyPairOptions<'pem', 'der'>,
        ): Promise<{ publicKey: string; privateKey: Buffer }>;
        function __promisify__(
            type: 'dsa',
            options: DSAKeyPairOptions<'der', 'pem'>,
        ): Promise<{ publicKey: Buffer; privateKey: string }>;
        function __promisify__(
            type: 'dsa',
            options: DSAKeyPairOptions<'der', 'der'>,
        ): Promise<{ publicKey: Buffer; privateKey: Buffer }>;

        function __promisify__(
            type: 'ec',
            options: ECKeyPairOptions<'pem', 'pem'>,
        ): Promise<{ publicKey: string; privateKey: string }>;
        function __promisify__(
            type: 'ec',
            options: ECKeyPairOptions<'pem', 'der'>,
        ): Promise<{ publicKey: string; privateKey: Buffer }>;
        function __promisify__(
            type: 'ec',
            options: ECKeyPairOptions<'der', 'pem'>,
        ): Promise<{ publicKey: Buffer; privateKey: string }>;
        function __promisify__(
            type: 'ec',
            options: ECKeyPairOptions<'der', 'der'>,
        ): Promise<{ publicKey: Buffer; privateKey: Buffer }>;
    }
}
