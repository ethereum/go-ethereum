import { ExecutionErrorDeploymentResult, PreviousRunErrorDeploymentResult, ReconciliationErrorDeploymentResult, ValidationErrorDeploymentResult } from "@nomicfoundation/ignition-core";
/**
 * Converts the result of an errored deployment into a message that can
 * be shown to the user in an exception.
 *
 * @param result - the errored deployment's result
 * @returns the text of the message
 */
export declare function errorDeploymentResultToExceptionMessage(result: ValidationErrorDeploymentResult | ReconciliationErrorDeploymentResult | ExecutionErrorDeploymentResult | PreviousRunErrorDeploymentResult): string;
//# sourceMappingURL=error-deployment-result-to-exception-message.d.ts.map