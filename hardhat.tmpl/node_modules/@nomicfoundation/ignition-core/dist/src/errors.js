"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.NomicIgnitionPluginError = exports.IgnitionPluginError = exports.IgnitionError = exports.CustomError = void 0;
const errors_list_1 = require("./internal/errors-list");
/**
 * Base error class extended by all custom errors.
 * Placeholder to allow us to customize error output formatting in the future.
 *
 * @beta
 */
class CustomError extends Error {
    constructor(message, cause) {
        super(message, cause !== undefined ? { cause } : undefined);
        this.name = this.constructor.name;
    }
}
exports.CustomError = CustomError;
/**
 * All exceptions intentionally thrown with Ignition-core
 * extend this class.
 *
 * @beta
 */
class IgnitionError extends CustomError {
    // We store the error descriptor as private field to avoid
    // interferring with Node's default error formatting.
    // We can use getters to access any private field without
    // interferring with it.
    //
    // Disabling this rule as private fields don't use `private`
    // eslint-disable-next-line @typescript-eslint/explicit-member-accessibility
    #errorDescriptor;
    constructor(errorDescriptor, messageArguments = {}, cause) {
        const prefix = `${(0, errors_list_1.getErrorCode)(errorDescriptor)}: `;
        const formattedMessage = _applyErrorMessageTemplate(errorDescriptor.message, messageArguments, false);
        super(prefix + formattedMessage, cause);
        this.#errorDescriptor = errorDescriptor;
    }
    get errorNumber() {
        return this.#errorDescriptor.number;
    }
}
exports.IgnitionError = IgnitionError;
/**
 * This class is used to throw errors from Ignition plugins made by third parties.
 *
 * @beta
 */
class IgnitionPluginError extends CustomError {
    static isIgnitionPluginError(error) {
        return (typeof error === "object" &&
            error !== null &&
            error._isIgnitionPluginError === true);
    }
    _isIgnitionPluginError = true;
    pluginName;
    constructor(pluginName, message, cause) {
        super(message, cause);
        this.pluginName = pluginName;
    }
}
exports.IgnitionPluginError = IgnitionPluginError;
/**
 * This class is used to throw errors from *core* Ignition plugins.
 * If you are developing a third-party plugin, use IgnitionPluginError instead.
 *
 * @beta
 */
class NomicIgnitionPluginError extends IgnitionPluginError {
    static isNomicIgnitionPluginError(error) {
        return (typeof error === "object" &&
            error !== null &&
            error._isNomicIgnitionPluginError === true);
    }
    _isNomicIgnitionPluginError = true;
}
exports.NomicIgnitionPluginError = NomicIgnitionPluginError;
function _applyErrorMessageTemplate(template, values, isRecursiveCall) {
    if (!isRecursiveCall) {
        for (const variableName of Object.keys(values)) {
            if (variableName.match(/^[a-zA-Z][a-zA-Z0-9]*$/) === null) {
                throw new IgnitionError(errors_list_1.ERRORS.INTERNAL.TEMPLATE_INVALID_VARIABLE_NAME, {
                    variable: variableName,
                });
            }
            const variableTag = `%${variableName}%`;
            if (!template.includes(variableTag)) {
                throw new IgnitionError(errors_list_1.ERRORS.INTERNAL.TEMPLATE_VARIABLE_NOT_FOUND, {
                    variable: variableName,
                });
            }
        }
    }
    if (template.includes("%%")) {
        return template
            .split("%%")
            .map((part) => _applyErrorMessageTemplate(part, values, true))
            .join("%");
    }
    for (const variableName of Object.keys(values)) {
        let value;
        if (values[variableName] === undefined) {
            value = "undefined";
        }
        else if (values[variableName] === null) {
            value = "null";
        }
        else {
            value = values[variableName].toString();
        }
        if (value === undefined) {
            value = "undefined";
        }
        const variableTag = `%${variableName}%`;
        if (value.match(/%([a-zA-Z][a-zA-Z0-9]*)?%/) !== null) {
            throw new IgnitionError(errors_list_1.ERRORS.INTERNAL.TEMPLATE_VALUE_CONTAINS_VARIABLE_TAG, { variable: variableName });
        }
        template = template.split(variableTag).join(value);
    }
    return template;
}
//# sourceMappingURL=errors.js.map