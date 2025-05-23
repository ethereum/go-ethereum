interface EtherscanGetSourceCodeNotOkResponse {
  status: "0";
  message: "NOTOK";
  result: string;
}

interface EtherscanGetSourceCodeOkResponse {
  status: "1";
  message: "OK";
  result: EtherscanContract[];
}

interface EtherscanContract {
  SourceCode: string;
  ABI: string;
  ContractName: string;
  CompilerVersion: string;
  OptimizationUsed: string;
  Runs: string;
  ConstructorArguments: string;
  EVMVersion: string;
  Library: string;
  LicenseType: string;
  Proxy: string;
  Implementation: string;
  SwarmSource: string;
}

export type EtherscanGetSourceCodeResponse =
  | EtherscanGetSourceCodeNotOkResponse
  | EtherscanGetSourceCodeOkResponse;

interface EtherscanVerifyNotOkResponse {
  status: "0";
  message: "NOTOK";
  result: string;
}

interface EtherscanVerifyOkResponse {
  status: "1";
  message: "OK";
  result: string;
}

export type EtherscanVerifyResponse =
  | EtherscanVerifyNotOkResponse
  | EtherscanVerifyOkResponse;
