declare module "string_decoder" {
    interface NodeStringDecoder {
        write(buffer: Buffer): string;
        end(buffer?: Buffer): string;
    }
    const StringDecoder: {
        new(encoding?: string): NodeStringDecoder;
    };
}
