import { requireNapiRsModule } from "../../../common/napi-rs";

const { linkHexStringBytecode } = requireNapiRsModule(
  "@nomicfoundation/edr"
) as typeof import("@nomicfoundation/edr");

export { linkHexStringBytecode };
