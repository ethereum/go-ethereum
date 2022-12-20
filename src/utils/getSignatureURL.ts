import { BINARIES_BASE_URL } from '../constants';

// only .sc files are being considered for signatures (https://github.com/ethereum/go-ethereum/blob/7519505d6fbd1fd29a8595aafbf880a04fb3e7e1/downloads.html#L299)
export const getSignatureURL = (filename: string) => `${BINARIES_BASE_URL}${filename}.asc`;
