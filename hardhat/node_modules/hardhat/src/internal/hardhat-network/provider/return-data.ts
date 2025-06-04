import { requireNapiRsModule } from "../../../common/napi-rs";

const { ReturnData } = requireNapiRsModule(
  "@nomicfoundation/edr"
) as typeof import("@nomicfoundation/edr");

export { ReturnData };
