import {
  getChecksum,
  getReleaseArch,
  getReleaseCommitHash,
  getReleaseCommitURL,
  getReleaseKind,
  getReleaseName,
  getReleaseSize,
  getReleaseURL,
  getSignatureURL
} from '.';

import { ReleaseData, ReleaseParams } from '../types';

export const mapReleasesData = ({ blobsList, isStableRelease }: ReleaseParams): ReleaseData[] => {
  return blobsList
    .filter(({ Name }: any) => !Name.endsWith('.asc') && !Name.endsWith('.sig')) // skip blobs we don't need to list
    .filter(({ Name }: any) =>
      isStableRelease ? !Name.includes('unstable') : Name.includes('unstable')
    ) // filter by stable/dev builds
    .map(({ Name, Properties }: any) => {
      const commitHash = getReleaseCommitHash(Name);

      return {
        release: {
          label: getReleaseName(Name),
          url: getReleaseURL(Name)
        },
        commit: {
          label: commitHash,
          url: getReleaseCommitURL(commitHash)
        },
        kind: getReleaseKind(Name),
        arch: getReleaseArch(Name),
        size: getReleaseSize(Properties['Content-Length']),
        // date is formatted later on the table, we use the raw value here for comparison
        published: Properties['Last-Modified'],
        signature: {
          label: 'Signature',
          url: getSignatureURL(Name)
        },
        checksum: getChecksum(Properties['Content-MD5'])
      };
    });
};
