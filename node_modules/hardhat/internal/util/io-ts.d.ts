import * as t from "io-ts";
export declare function optionalOrNullable<TypeT, OutputT, InputT>(codec: t.Type<TypeT, OutputT, InputT>, name?: string): t.Type<TypeT | undefined, OutputT | undefined, InputT | undefined | null>;
export declare function optional<TypeT, OutputT, InputT>(codec: t.Type<TypeT, OutputT, InputT>, name?: string): t.Type<TypeT | undefined, OutputT | undefined, InputT | undefined>;
export declare function nullable<TypeT, OutputT, InputT>(codec: t.Type<TypeT, OutputT, InputT>, name?: string): t.Type<TypeT | null, OutputT | null, InputT | null>;
//# sourceMappingURL=io-ts.d.ts.map