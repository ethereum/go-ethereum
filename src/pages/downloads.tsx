import {
  Code,
  Link,
  ListItem,
  Stack,
  Text,
  UnorderedList,
} from '@chakra-ui/react';
import type { NextPage } from 'next';

import { DownloadsHero, DownloadsSection } from '../components/UI/downloads';


const DownloadsPage: NextPage = ({}) => {
  return (
    <>
     {/* TODO: add PageMetadata */}
     
     <main>
      <Stack spacing={4}>
        {/* TODO: replace hardcoded strings with build information */}
        <DownloadsHero
          currentBuildName={'Sentry Omega'}
          currentBuildVersion={'v1.10.23'}
          linuxBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-linux-amd64-1.10.25-69568c55.tar.gz'}
          macOSBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-darwin-amd64-1.10.25-69568c55.tar.gz'}
          releaseNotesURL={''}
          sourceCodeURL={'https://github.com/ethereum/go-ethereum/archive/v1.10.25.tar.gz'}
          windowsBuildURL={'https://gethstore.blob.core.windows.net/builds/geth-windows-amd64-1.10.25-69568c55.exe'}
        />

        <DownloadsSection
          imgSrc='/images/pages/gopher-home-side-desktop.svg'
          imgAltText='Gopher facing right'
          sectionTitle='Specific Versions'
        >
          <Stack p={4}>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              If you&apos;re looking for a specific release, operating system or architecture, below you will find:
            </Text>

            <UnorderedList px={4}>
              <ListItem>
                <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
                  All stable and develop builds of Geth and tools
                </Text>
              </ListItem>
              <ListItem>
                <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
                  Archives for non-primary processor architectures
                </Text>
              </ListItem>
              <ListItem>
                <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
                  Android library archives and iOS XCode frameworks
                </Text>
              </ListItem>
            </UnorderedList>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Please select your desired platform from the lists below and download your bundle of choice. Please be aware that the MD5 checksums are provided by our binary hosting platform (Azure Blobstore) to help check for download errors. For security guarantees please verify any downloads via the attached PGP signature files (see{' '}
              <Link
                href={''}
                isExternal
                textDecoration='underline'
                color='#11866f'
                _hover={{ color: '#1d242c', textDecorationColor: '#1d242c' }}
                _focus={{
                  color: '#11866f',
                  boxShadow: '0 0 0 1px #11866f !important',
                  textDecoration: 'none'
                }}
                _pressed={{ color: '#25453f', textDecorationColor: '#25453f' }}
              >
                OpenPGP
              </Link>{' '}
              Signatures for details).
            </Text>
          </Stack>
        </DownloadsSection>

        <DownloadsSection sectionTitle='Importing keys and verifying builds'>
          <Stack p={4} borderBottom='2px solid #11866f'>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              You can import the build server public keys by grabbing the individual keys directly from the keyserver network:
            </Text>
            <Code p={4}>
              gpg --recv-keys F9585DE6 C2FF8BBF 9BA28146 7B9E2481 D2A67EAC
            </Code>
          </Stack>

          <Stack p={4} borderBottom='2px solid #11866f'>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Similarly you can import all the developer public keys by grabbing them directly from the keyserver network:
            </Text>

            <Code p={4}>
              gpg --recv-keys E058A81C  05A5DDF0 1CCB7DD2
            </Code>
          </Stack>

          <Stack p={4}>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              From the download listings above you should see a link both to the downloadable archives as well as detached signature files. To verify the authenticity of any downloaded data, grab both files and then run:
            </Text>

            <Code p={4}>
              gpg --verify geth-linux-amd64-1.5.0-d0c820ac.tar.gz.asc
            </Code>
          </Stack>
        </DownloadsSection>
      </Stack>
     </main>
    </>
  )
}

export default DownloadsPage