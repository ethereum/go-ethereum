"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ModuleParameterRuntimeValueImplementation = exports.AccountRuntimeValueImplementation = exports.IgnitionModuleImplementation = exports.SendDataFutureImplementation = exports.ReadEventArgumentFutureImplementation = exports.ArtifactContractAtFutureImplementation = exports.NamedContractAtFutureImplementation = exports.NamedEncodeFunctionCallFutureImplementation = exports.NamedStaticCallFutureImplementation = exports.NamedContractCallFutureImplementation = exports.ArtifactLibraryDeploymentFutureImplementation = exports.NamedLibraryDeploymentFutureImplementation = exports.ArtifactContractDeploymentFutureImplementation = exports.NamedContractDeploymentFutureImplementation = void 0;
const module_1 = require("../types/module");
const customInspectSymbol = Symbol.for("util.inspect.custom");
class BaseFutureImplementation {
    id;
    type;
    module;
    dependencies = new Set();
    constructor(id, type, module) {
        this.id = id;
        this.type = type;
        this.module = module;
    }
    [customInspectSymbol](_depth, { inspect }) {
        const padding = " ".repeat(2);
        return `Future ${this.id} {
    Type: ${module_1.FutureType[this.type]}
    Module: ${this.module.id}
    Dependencies: ${inspect(Array.from(this.dependencies).map((f) => f.id)).replace(/\n/g, `\n${padding}`)}
  }`;
    }
}
class NamedContractDeploymentFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    constructorArgs;
    libraries;
    value;
    from;
    constructor(id, module, contractName, constructorArgs, libraries, value, from) {
        super(id, module_1.FutureType.NAMED_ARTIFACT_CONTRACT_DEPLOYMENT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.constructorArgs = constructorArgs;
        this.libraries = libraries;
        this.value = value;
        this.from = from;
    }
}
exports.NamedContractDeploymentFutureImplementation = NamedContractDeploymentFutureImplementation;
class ArtifactContractDeploymentFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    constructorArgs;
    artifact;
    libraries;
    value;
    from;
    constructor(id, module, contractName, constructorArgs, artifact, libraries, value, from) {
        super(id, module_1.FutureType.CONTRACT_DEPLOYMENT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.constructorArgs = constructorArgs;
        this.artifact = artifact;
        this.libraries = libraries;
        this.value = value;
        this.from = from;
    }
}
exports.ArtifactContractDeploymentFutureImplementation = ArtifactContractDeploymentFutureImplementation;
class NamedLibraryDeploymentFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    libraries;
    from;
    constructor(id, module, contractName, libraries, from) {
        super(id, module_1.FutureType.NAMED_ARTIFACT_LIBRARY_DEPLOYMENT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.libraries = libraries;
        this.from = from;
    }
}
exports.NamedLibraryDeploymentFutureImplementation = NamedLibraryDeploymentFutureImplementation;
class ArtifactLibraryDeploymentFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    artifact;
    libraries;
    from;
    constructor(id, module, contractName, artifact, libraries, from) {
        super(id, module_1.FutureType.LIBRARY_DEPLOYMENT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.artifact = artifact;
        this.libraries = libraries;
        this.from = from;
    }
}
exports.ArtifactLibraryDeploymentFutureImplementation = ArtifactLibraryDeploymentFutureImplementation;
class NamedContractCallFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    functionName;
    contract;
    args;
    value;
    from;
    constructor(id, module, functionName, contract, args, value, from) {
        super(id, module_1.FutureType.CONTRACT_CALL, module);
        this.id = id;
        this.module = module;
        this.functionName = functionName;
        this.contract = contract;
        this.args = args;
        this.value = value;
        this.from = from;
    }
}
exports.NamedContractCallFutureImplementation = NamedContractCallFutureImplementation;
class NamedStaticCallFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    functionName;
    contract;
    args;
    nameOrIndex;
    from;
    constructor(id, module, functionName, contract, args, nameOrIndex, from) {
        super(id, module_1.FutureType.STATIC_CALL, module);
        this.id = id;
        this.module = module;
        this.functionName = functionName;
        this.contract = contract;
        this.args = args;
        this.nameOrIndex = nameOrIndex;
        this.from = from;
    }
}
exports.NamedStaticCallFutureImplementation = NamedStaticCallFutureImplementation;
class NamedEncodeFunctionCallFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    functionName;
    contract;
    args;
    constructor(id, module, functionName, contract, args) {
        super(id, module_1.FutureType.ENCODE_FUNCTION_CALL, module);
        this.id = id;
        this.module = module;
        this.functionName = functionName;
        this.contract = contract;
        this.args = args;
    }
}
exports.NamedEncodeFunctionCallFutureImplementation = NamedEncodeFunctionCallFutureImplementation;
class NamedContractAtFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    address;
    constructor(id, module, contractName, address) {
        super(id, module_1.FutureType.NAMED_ARTIFACT_CONTRACT_AT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.address = address;
    }
}
exports.NamedContractAtFutureImplementation = NamedContractAtFutureImplementation;
class ArtifactContractAtFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    contractName;
    address;
    artifact;
    constructor(id, module, contractName, address, artifact) {
        super(id, module_1.FutureType.CONTRACT_AT, module);
        this.id = id;
        this.module = module;
        this.contractName = contractName;
        this.address = address;
        this.artifact = artifact;
    }
}
exports.ArtifactContractAtFutureImplementation = ArtifactContractAtFutureImplementation;
class ReadEventArgumentFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    futureToReadFrom;
    eventName;
    nameOrIndex;
    emitter;
    eventIndex;
    constructor(id, module, futureToReadFrom, eventName, nameOrIndex, emitter, eventIndex) {
        super(id, module_1.FutureType.READ_EVENT_ARGUMENT, module);
        this.id = id;
        this.module = module;
        this.futureToReadFrom = futureToReadFrom;
        this.eventName = eventName;
        this.nameOrIndex = nameOrIndex;
        this.emitter = emitter;
        this.eventIndex = eventIndex;
    }
}
exports.ReadEventArgumentFutureImplementation = ReadEventArgumentFutureImplementation;
class SendDataFutureImplementation extends BaseFutureImplementation {
    id;
    module;
    to;
    value;
    data;
    from;
    constructor(id, module, to, value, data, from) {
        super(id, module_1.FutureType.SEND_DATA, module);
        this.id = id;
        this.module = module;
        this.to = to;
        this.value = value;
        this.data = data;
        this.from = from;
    }
}
exports.SendDataFutureImplementation = SendDataFutureImplementation;
class IgnitionModuleImplementation {
    id;
    results;
    futures = new Set();
    submodules = new Set();
    constructor(id, results) {
        this.id = id;
        this.results = results;
    }
    [customInspectSymbol](_depth, { inspect }) {
        const padding = " ".repeat(2);
        return `IgnitionModule ${this.id} {
    Futures: ${inspect(this.futures).replace(/\n/g, `\n${padding}`)}
    Results: ${inspect(this.results).replace(/\n/g, `\n${padding}`)}
    Submodules: ${inspect(Array.from(this.submodules).map((m) => m.id)).replace(/\n/g, `\n${padding}`)}
  }`;
    }
}
exports.IgnitionModuleImplementation = IgnitionModuleImplementation;
class AccountRuntimeValueImplementation {
    accountIndex;
    type = module_1.RuntimeValueType.ACCOUNT;
    constructor(accountIndex) {
        this.accountIndex = accountIndex;
    }
    [customInspectSymbol](_depth, _inspectOptions) {
        return `Account RuntimeValue {
    accountIndex: ${this.accountIndex}
}`;
    }
}
exports.AccountRuntimeValueImplementation = AccountRuntimeValueImplementation;
class ModuleParameterRuntimeValueImplementation {
    moduleId;
    name;
    defaultValue;
    type = module_1.RuntimeValueType.MODULE_PARAMETER;
    constructor(moduleId, name, defaultValue) {
        this.moduleId = moduleId;
        this.name = name;
        this.defaultValue = defaultValue;
    }
    [customInspectSymbol](_depth, { inspect }) {
        return `Module Parameter RuntimeValue {
    name: ${this.name}${this.defaultValue !== undefined
            ? `
    default value: ${inspect(this.defaultValue)}`
            : ""}
}`;
    }
}
exports.ModuleParameterRuntimeValueImplementation = ModuleParameterRuntimeValueImplementation;
//# sourceMappingURL=module.js.map