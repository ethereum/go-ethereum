import { BINARIES_BASE_URL } from '../constants';

export const getReleaseURL = (filename: string) => `${BINARIES_BASE_URL}${filename}`;
