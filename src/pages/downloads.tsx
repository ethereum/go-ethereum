import { Stack } from '@chakra-ui/react';
import type { NextPage } from 'next';

import { DownloadsHero } from '../components/UI/homepage';

import {

} from '../constants';

const DownloadsPage: NextPage = ({}) => {
  return (
    <>
     {/* TODO: add PageMetadata */}
     
     <main>
      <Stack spacing={4}>
        <DownloadsHero
          currentBuildName={'Sentry Omega'}
          currentBuildVersion={'v1.10.23'}
          linuxBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.10.25-69568c55.tar.gz'}
          macOSBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-darwin-amd64-1.10.25-69568c55.tar.gz'}
          releaseNotesURL={''}
          sourceCodeURL={'https://github.com/ethereum/go-ethereum/archive/v1.10.25.tar.gz'}
          windowsBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-windows-amd64-1.10.25-69568c55.exe'}
        />
        <p>Hello</p>
      </Stack>
     </main>
    </>
  )
}

export default DownloadsPage