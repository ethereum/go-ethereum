"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EVENT_IMPORTS = exports.EVENT_METHOD_OVERRIDES = exports.generateGetEventForContract = exports.generateTypedContractEvent = exports.generateGetEventForInterface = exports.generateEventNameOrSignature = exports.generateEventArgType = exports.generateEventInputs = exports.generateEventSignature = exports.generateInterfaceEventDescription = exports.generateEventTypeExport = exports.generateEventTypeExports = exports.generateEventFilters = void 0;
/* eslint-disable import/no-extraneous-dependencies */
const typechain_1 = require("typechain");
const types_1 = require("./types");
function generateEventFilters(events) {
    if (events.length === 1) {
        const event = events[0];
        const typedEventFilter = generateTypedContractEvent(event, false);
        return `
      '${generateEventSignature(event)}': ${typedEventFilter};
      ${event.name}: ${typedEventFilter};
    `;
    }
    else {
        return events
            .map((event) => `'${generateEventSignature(event)}': ${generateTypedContractEvent(event, true)};`)
            .join('\n');
    }
}
exports.generateEventFilters = generateEventFilters;
function generateEventTypeExports(events) {
    if (events.length === 1) {
        return generateEventTypeExport(events[0], false);
    }
    else {
        return events.map((e) => generateEventTypeExport(e, true)).join('\n');
    }
}
exports.generateEventTypeExports = generateEventTypeExports;
function generateEventTypeExport(event, includeArgTypes) {
    const components = event.inputs.map((input, i) => { var _a; return ({ name: (_a = input.name) !== null && _a !== void 0 ? _a : `arg${i.toString()}`, type: input.type }); });
    const inputTuple = (0, types_1.generateInputComplexTypeAsTuple)(components, {
        useStructs: true,
        includeLabelsInTupleTypes: true,
    });
    const outputTuple = (0, types_1.generateOutputComplexTypeAsTuple)(components, {
        useStructs: true,
        includeLabelsInTupleTypes: true,
    });
    const outputObject = (0, types_1.generateOutputComplexTypesAsObject)(components, { useStructs: true }) || '{}';
    const identifier = generateEventIdentifier(event, { includeArgTypes });
    return `
    export namespace ${identifier} {
      export type InputTuple = ${inputTuple};
      export type OutputTuple = ${outputTuple};
      export interface OutputObject ${outputObject};
      export type Event = TypedContractEvent<InputTuple, OutputTuple, OutputObject>
      export type Filter = TypedDeferredTopicFilter<Event>
      export type Log = TypedEventLog<Event>
      export type LogDescription = TypedLogDescription<Event>
    }

  `;
}
exports.generateEventTypeExport = generateEventTypeExport;
function generateInterfaceEventDescription(event) {
    return `'${generateEventSignature(event)}': EventFragment;`;
}
exports.generateInterfaceEventDescription = generateInterfaceEventDescription;
function generateEventSignature(event) {
    return `${event.name}(${event.inputs.map((input) => input.type.originalType).join(',')})`;
}
exports.generateEventSignature = generateEventSignature;
function generateEventInputs(eventArgs) {
    if (eventArgs.length === 0) {
        return '';
    }
    return (eventArgs
        .map((arg, index) => {
        return `${arg.name ? (0, typechain_1.createPositionalIdentifier)(arg.name) : `arg${index}`}?: ${generateEventArgType(arg)}`;
    })
        .join(', ') + ', ');
}
exports.generateEventInputs = generateEventInputs;
function generateEventArgType(eventArg) {
    return eventArg.isIndexed ? `${(0, types_1.generateInputType)({ useStructs: true }, eventArg.type)} | null` : 'null';
}
exports.generateEventArgType = generateEventArgType;
function generateEventNameOrSignature(event, useSignature) {
    return useSignature ? generateEventSignature(event) : event.name;
}
exports.generateEventNameOrSignature = generateEventNameOrSignature;
// export function generateGetEventForInterface(event: EventDeclaration, useSignature: boolean): string {
//   return `getEvent(nameOrSignatureOrTopic: '${
//     useSignature ? generateEventSignature(event) : event.name
//   }'): EventFragment;`
// }
function generateGetEventForInterface(args) {
    if (args.length === 0)
        return '';
    return `getEvent(nameOrSignatureOrTopic: ${args.map((s) => `"${s}"`).join(' | ')}): EventFragment;`;
}
exports.generateGetEventForInterface = generateGetEventForInterface;
function generateTypedContractEvent(event, useSignature) {
    const eventIdentifier = generateEventIdentifier(event, {
        includeArgTypes: useSignature,
    });
    return `TypedContractEvent<${eventIdentifier}.InputTuple, ${eventIdentifier}.OutputTuple, ${eventIdentifier}.OutputObject>`;
}
exports.generateTypedContractEvent = generateTypedContractEvent;
function generateGetEventForContract(event, useSignature) {
    const typedContractEvent = generateTypedContractEvent(event, useSignature);
    return `getEvent(key: '${useSignature ? generateEventSignature(event) : event.name}'): ${typedContractEvent};`;
}
exports.generateGetEventForContract = generateGetEventForContract;
function generateEventIdentifier(event, { includeArgTypes } = {}) {
    if (includeArgTypes) {
        return (0, typechain_1.getFullSignatureAsSymbolForEvent)(event) + '_Event';
    }
    else {
        return event.name + 'Event';
    }
}
exports.EVENT_METHOD_OVERRIDES = `
  queryFilter<TCEvent extends TypedContractEvent>(
    event: TCEvent,
    fromBlockOrBlockhash?: string | number | undefined,
    toBlock?: string | number | undefined,
  ): Promise<Array<TypedEventLog<TCEvent>>>
  queryFilter<TCEvent extends TypedContractEvent>(
    filter: TypedDeferredTopicFilter<TCEvent>,
    fromBlockOrBlockhash?: string | number | undefined,
    toBlock?: string | number | undefined
  ): Promise<Array<TypedEventLog<TCEvent>>>;

  on<TCEvent extends TypedContractEvent>(event: TCEvent, listener: TypedListener<TCEvent>): Promise<this>
  on<TCEvent extends TypedContractEvent>(filter: TypedDeferredTopicFilter<TCEvent>, listener: TypedListener<TCEvent>): Promise<this>
  
  once<TCEvent extends TypedContractEvent>(event: TCEvent, listener: TypedListener<TCEvent>): Promise<this>
  once<TCEvent extends TypedContractEvent>(filter: TypedDeferredTopicFilter<TCEvent>, listener: TypedListener<TCEvent>): Promise<this>

  listeners<TCEvent extends TypedContractEvent>(
    event: TCEvent
  ): Promise<Array<TypedListener<TCEvent>>>;
  listeners(eventName?: string): Promise<Array<Listener>>
  removeAllListeners<TCEvent extends TypedContractEvent>(event?: TCEvent): Promise<this>
`;
exports.EVENT_IMPORTS = [
    'TypedContractEvent',
    'TypedDeferredTopicFilter',
    'TypedEventLog',
    'TypedLogDescription',
    'TypedListener',
];
//# sourceMappingURL=events.js.map