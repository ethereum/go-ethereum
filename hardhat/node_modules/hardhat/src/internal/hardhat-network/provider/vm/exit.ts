import { requireNapiRsModule } from "../../../../common/napi-rs";

const { ExitCode } = requireNapiRsModule(
  "@nomicfoundation/edr"
) as typeof import("@nomicfoundation/edr");

export { ExitCode };
