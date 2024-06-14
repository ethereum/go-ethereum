import { IgnitionModule, IgnitionModuleResult } from "./types/module";
import { SerializedIgnitionModule } from "./types/serialization";
/**
 * Serialize an Ignition module.
 *
 * @beta
 */
export declare class IgnitionModuleSerializer {
    static serialize(ignitionModule: IgnitionModule<string, string, IgnitionModuleResult<string>>): SerializedIgnitionModule;
    private static _serializeModule;
    private static _serializeFuture;
    private static _convertLibrariesToLibraryTokens;
    private static _serializeAccountRuntimeValue;
    private static _serializeModuleParamterRuntimeValue;
    private static _serializeBigint;
    private static _jsonStringifyWithBigint;
    private static _convertFutureToFutureToken;
    private static _convertModuleToModuleToken;
    private static _getModulesAndSubmoduleFor;
}
/**
 * Deserialize an `IgnitionModule` that was previously serialized using
 * IgnitionModuleSerializer.
 *
 * @beta
 */
export declare class IgnitionModuleDeserializer {
    static deserialize(serializedIgnitionModule: SerializedIgnitionModule): IgnitionModule<string, string, IgnitionModuleResult<string>>;
    private static _getSerializedModulesInReverseTopologicalOrder;
    private static _getSerializedFuturesInReverseTopologicalOrder;
    private static _deserializeArgument;
    private static _deserializedBigint;
    private static _jsonParseWithBigint;
    private static _isSerializedFutureToken;
    private static _isSerializedBigInt;
    private static _getAllFuturesFor;
    private static _deserializeFuture;
    private static _lookup;
    private static _isSerializedAccountRuntimeValue;
    private static _deserializeAccountRuntimeValue;
    private static _isSerializedModuleParameterRuntimeValue;
    private static _deserializeModuleParameterRuntimeValue;
}
//# sourceMappingURL=ignition-module-serializer.d.ts.map