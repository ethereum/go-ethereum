import { Integration, Options } from '@sentry/types';
export declare const installedIntegrations: string[];
/** Map of integrations assigned to a client */
export interface IntegrationIndex {
    [key: string]: Integration;
}
/** Gets integration to install */
export declare function getIntegrationsToSetup(options: Options): Integration[];
/** Setup given integration */
export declare function setupIntegration(integration: Integration): void;
/**
 * Given a list of integration instances this installs them all. When `withDefaults` is set to `true` then all default
 * integrations are added unless they were already provided before.
 * @param integrations array of integration instances
 * @param withDefault should enable default integrations
 */
export declare function setupIntegrations<O extends Options>(options: O): IntegrationIndex;
//# sourceMappingURL=integration.d.ts.map