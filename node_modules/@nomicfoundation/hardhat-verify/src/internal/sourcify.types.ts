export enum ContractStatus {
  PERFECT = "perfect",
  PARTIAL = "partial",
  NOT_FOUND = "false",
}

interface SourcifyIsVerifiedNotOkResponse {
  address: string;
  status: ContractStatus.NOT_FOUND;
}

interface SourcifyIsVerifiedOkResponse {
  address: string;
  chainIds: Array<{
    chainId: string;
    status: ContractStatus.PERFECT | ContractStatus.PARTIAL;
  }>;
}

export type SourcifyIsVerifiedResponse =
  | SourcifyIsVerifiedNotOkResponse
  | SourcifyIsVerifiedOkResponse;

interface SourcifyVerifyNotOkResponse {
  error: string;
}

interface SourcifyVerifyOkResponse {
  result: Array<{
    address: string;
    chainId: string;
    status: string;
    message?: string;
    libraryMap?: Record<string, string>;
  }>;
}

export type SourcifyVerifyResponse =
  | SourcifyVerifyNotOkResponse
  | SourcifyVerifyOkResponse;
