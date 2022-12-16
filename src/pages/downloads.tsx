import { Code, Flex, Link, ListItem, Stack, Text, UnorderedList } from '@chakra-ui/react';
import type { GetStaticProps, NextPage } from 'next';
import { useState } from 'react';
import { XMLParser } from 'fast-xml-parser';

import {
  DownloadsHero,
  DownloadsSection,
  DownloadsTable,
  SpecificVersionsSection
} from '../components/UI/downloads';
import { DataTable, PageMetadata } from '../components/UI';

import {
  DEFAULT_BUILD_AMOUNT_TO_SHOW,
  DOWNLOADS_OPENPGP_BUILD_HEADERS,
  DOWNLOADS_OPENPGP_DEVELOPER_HEADERS,
  GETH_REPO_URL,
  METADATA,
  LATEST_SOURCES_BASE_URL,
  RELEASE_NOTES_BASE_URL,
  DOWNLOADS_OPENPGP_SIGNATURES,
  DOWNLOADS_DEVELOPERS_DATA
} from '../constants';

import {
  fetchLatestReleaseCommit,
  fetchLatestReleaseVersionAndName,
  fetchXMLData,
  getLatestBinaryURL,
  getSortedReleases,
  mapReleasesData
} from '../utils';
import { LatestReleasesData, ReleaseData } from '../types';

// This function gets called at build time on server-side.
// It'll be called again if a new request comes in after 1hr, so data is refreshed periodically
// More info: https://nextjs.org/docs/basic-features/data-fetching/incremental-static-regeneration
export const getStaticProps: GetStaticProps = async () => {
  // ==== LATEST RELEASES DATA ====

  // Latest version number & release name
  const { versionNumber, releaseName } = await fetchLatestReleaseVersionAndName();
  // Latest release commit hash
  const commit = await fetchLatestReleaseCommit(versionNumber);

  // Latest binaries urls
  const LATEST_LINUX_BINARY_URL = getLatestBinaryURL('linux', versionNumber, commit);
  const LATEST_MACOS_BINARY_URL = getLatestBinaryURL('darwin', versionNumber, commit);
  const LATEST_WINDOWS_BINARY_URL = getLatestBinaryURL('windows', versionNumber, commit);

  // Sources urls
  const LATEST_SOURCES_URL = `${LATEST_SOURCES_BASE_URL}${versionNumber}.tar.gz`;
  const RELEASE_NOTES_URL = `${RELEASE_NOTES_BASE_URL}${versionNumber}`;

  const LATEST_RELEASES_DATA = {
    versionNumber,
    releaseName,
    urls: {
      LATEST_LINUX_BINARY_URL,
      LATEST_MACOS_BINARY_URL,
      LATEST_WINDOWS_BINARY_URL,
      LATEST_SOURCES_URL,
      RELEASE_NOTES_URL
    }
  };

  // ==== ALL RELEASES DATA ====

  // 1) fetch XML data
  try {
    const [
      ALL_LINUX_RELEASES_XML_DATA,
      ALL_LINUX_ALL_TOOLS_RELEASES_XML_DATA,
      ALL_MACOS_RELEASES_XML_DATA,
      ALL_MACOS_ALL_TOOLS_RELEASES_XML_DATA,
      ALL_WINDOWS_RELEASES_XML_DATA,
      ALL_WINDOWS_ALL_TOOLS_RELEASES_XML_DATA,
      ALL_ANDROID_RELEASES_XML_DATA,
      ALL_IOS_RELEASES_XML_DATA
    ] = await fetchXMLData();

    // 2) XML data parsing
    const parser = new XMLParser();

    // linux
    const linuxJson = parser.parse(ALL_LINUX_RELEASES_XML_DATA);
    const ALL_LINUX_BLOBS_JSON_DATA = linuxJson.EnumerationResults.Blobs.Blob;

    const linuxAllToolsJson = parser.parse(ALL_LINUX_ALL_TOOLS_RELEASES_XML_DATA);
    const ALL_LINUX_ALL_TOOLS_BLOBS_JSON_DATA = linuxAllToolsJson.EnumerationResults.Blobs.Blob;

    // macOS
    const macOSJson = parser.parse(ALL_MACOS_RELEASES_XML_DATA);
    const ALL_MACOS_BLOBS_JSON_DATA = macOSJson.EnumerationResults.Blobs.Blob;

    const macOSAllToolsJson = parser.parse(ALL_MACOS_ALL_TOOLS_RELEASES_XML_DATA);
    const ALL_MACOS_ALL_TOOLS_BLOBS_JSON_DATA = macOSAllToolsJson.EnumerationResults.Blobs.Blob;

    // windows
    const windowsJson = parser.parse(ALL_WINDOWS_RELEASES_XML_DATA);
    const ALL_WINDOWS_BLOBS_JSON_DATA = windowsJson.EnumerationResults.Blobs.Blob;

    const windowsAllToolsJson = parser.parse(ALL_WINDOWS_ALL_TOOLS_RELEASES_XML_DATA);
    const ALL_WINDOWS_ALL_TOOLS_BLOBS_JSON_DATA = windowsAllToolsJson.EnumerationResults.Blobs.Blob;

    // android
    const androidJson = parser.parse(ALL_ANDROID_RELEASES_XML_DATA);
    const ALL_ANDROID_BLOBS_JSON_DATA = androidJson.EnumerationResults.Blobs.Blob;

    // iOS
    const iOSJson = parser.parse(ALL_IOS_RELEASES_XML_DATA);
    const ALL_IOS_BLOBS_JSON_DATA = iOSJson.EnumerationResults.Blobs.Blob;

    // 3) get blobs
    // linux
    const LINUX_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_LINUX_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const LINUX_ALLTOOLS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_LINUX_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const LINUX_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_LINUX_BLOBS_JSON_DATA,
      isStableRelease: false
    });
    const LINUX_ALLTOOLS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_LINUX_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: false
    });

    // macOS
    const MACOS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_MACOS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const MACOS_ALLTOOLS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_MACOS_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const MACOS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_MACOS_BLOBS_JSON_DATA,
      isStableRelease: false
    });
    const MACOS_ALLTOOLS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_MACOS_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: false
    });

    // windows
    const WINDOWS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_WINDOWS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const WINDOWS_ALLTOOLS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_WINDOWS_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const WINDOWS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_WINDOWS_BLOBS_JSON_DATA,
      isStableRelease: false
    });
    const WINDOWS_ALLTOOLS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_WINDOWS_ALL_TOOLS_BLOBS_JSON_DATA,
      isStableRelease: false
    });

    // android
    const ANDROID_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_ANDROID_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const ANDROID_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_ANDROID_BLOBS_JSON_DATA,
      isStableRelease: false
    });

    // iOS
    const IOS_STABLE_RELEASES_DATA = mapReleasesData({
      blobsList: ALL_IOS_BLOBS_JSON_DATA,
      isStableRelease: true
    });
    const IOS_DEV_BUILDS_DATA = mapReleasesData({
      blobsList: ALL_IOS_BLOBS_JSON_DATA,
      isStableRelease: false
    });

    return {
      props: {
        data: {
          // latest
          LATEST_RELEASES_DATA,
          // linux
          ALL_LINUX_STABLE_RELEASES: getSortedReleases(
            LINUX_STABLE_RELEASES_DATA,
            LINUX_ALLTOOLS_STABLE_RELEASES_DATA
          ),
          ALL_LINUX_DEV_BUILDS: getSortedReleases(
            LINUX_DEV_BUILDS_DATA,
            LINUX_ALLTOOLS_DEV_BUILDS_DATA
          ),
          // macOS
          ALL_MACOS_STABLE_RELEASES: getSortedReleases(
            MACOS_STABLE_RELEASES_DATA,
            MACOS_ALLTOOLS_STABLE_RELEASES_DATA
          ),
          ALL_MACOS_DEV_BUILDS: getSortedReleases(
            MACOS_DEV_BUILDS_DATA,
            MACOS_ALLTOOLS_DEV_BUILDS_DATA
          ),
          // windows
          ALL_WINDOWS_STABLE_RELEASES: getSortedReleases(
            WINDOWS_STABLE_RELEASES_DATA,
            WINDOWS_ALLTOOLS_STABLE_RELEASES_DATA
          ),
          ALL_WINDOWS_DEV_BUILDS: getSortedReleases(
            WINDOWS_DEV_BUILDS_DATA,
            WINDOWS_ALLTOOLS_DEV_BUILDS_DATA
          ),
          // android
          ALL_ANDROID_STABLE_RELEASES: getSortedReleases(ANDROID_STABLE_RELEASES_DATA),
          ALL_ANDROID_DEV_BUILDS: getSortedReleases(ANDROID_DEV_BUILDS_DATA),
          // iOS
          ALL_IOS_STABLE_RELEASES: getSortedReleases(IOS_STABLE_RELEASES_DATA),
          ALL_IOS_DEV_BUILDS: getSortedReleases(IOS_DEV_BUILDS_DATA)
        }
      },
      // using ISR here (https://nextjs.org/docs/basic-features/data-fetching/incremental-static-regeneration)
      revalidate: 3600 // 1hr in seconds
    };
  } catch (error) {
    console.error(error);

    return { notFound: true };
  }
};

interface Props {
  data: {
    // latest
    LATEST_RELEASES_DATA: LatestReleasesData;
    // linux
    ALL_LINUX_STABLE_RELEASES: ReleaseData[];
    ALL_LINUX_DEV_BUILDS: ReleaseData[];
    // macOS
    ALL_MACOS_STABLE_RELEASES: ReleaseData[];
    ALL_MACOS_DEV_BUILDS: ReleaseData[];
    // windows
    ALL_WINDOWS_STABLE_RELEASES: ReleaseData[];
    ALL_WINDOWS_DEV_BUILDS: ReleaseData[];
    // android
    ALL_ANDROID_STABLE_RELEASES: ReleaseData[];
    ALL_ANDROID_DEV_BUILDS: ReleaseData[];
    // iOS
    ALL_IOS_STABLE_RELEASES: ReleaseData[];
    ALL_IOS_DEV_BUILDS: ReleaseData[];
  };
}

const DownloadsPage: NextPage<Props> = ({ data }) => {
  const {
    // latest
    LATEST_RELEASES_DATA,
    // linux
    ALL_LINUX_STABLE_RELEASES,
    ALL_LINUX_DEV_BUILDS,
    // macOS
    ALL_MACOS_STABLE_RELEASES,
    ALL_MACOS_DEV_BUILDS,
    // windows
    ALL_WINDOWS_STABLE_RELEASES,
    ALL_WINDOWS_DEV_BUILDS,
    // android
    ALL_ANDROID_STABLE_RELEASES,
    ALL_ANDROID_DEV_BUILDS,
    // iOS
    ALL_IOS_STABLE_RELEASES,
    ALL_IOS_DEV_BUILDS
  } = data;

  const [amountStableReleases, setAmountStableReleases] = useState(DEFAULT_BUILD_AMOUNT_TO_SHOW);
  const [amountDevBuilds, setAmountDevBuilds] = useState(DEFAULT_BUILD_AMOUNT_TO_SHOW);

  const [totalStableReleases, setTotalStableReleases] = useState(ALL_LINUX_STABLE_RELEASES.length);
  const [totalDevBuilds, setTotalDevBuilds] = useState(ALL_LINUX_DEV_BUILDS.length);

  const showMoreStableReleases = () => {
    setAmountStableReleases(amountStableReleases + 12);
  };

  const showMoreDevBuilds = () => {
    setAmountDevBuilds(amountDevBuilds + 12);
  };

  return (
    <>
      <PageMetadata title={METADATA.DOWNLOADS_TITLE} description={METADATA.DOWNLOADS_DESCRIPTION} />

      <main id='main-content'>
        <Stack spacing={{ base: 4, lg: 8 }}>
          <DownloadsHero
            currentBuild={LATEST_RELEASES_DATA.releaseName}
            currentBuildVersion={LATEST_RELEASES_DATA.versionNumber}
            linuxBuildURL={LATEST_RELEASES_DATA.urls.LATEST_LINUX_BINARY_URL}
            macOSBuildURL={LATEST_RELEASES_DATA.urls.LATEST_MACOS_BINARY_URL}
            windowsBuildURL={LATEST_RELEASES_DATA.urls.LATEST_WINDOWS_BINARY_URL}
            sourceCodeURL={LATEST_RELEASES_DATA.urls.LATEST_SOURCES_URL}
            releaseNotesURL={LATEST_RELEASES_DATA.urls.RELEASE_NOTES_URL}
          />

          <SpecificVersionsSection>
            <Stack p={4}>
              <Text textStyle='quick-link-text'>
                If you&apos;re looking for a specific release, operating system or architecture,
                below you will find:
              </Text>

              <UnorderedList px={4}>
                <ListItem>
                  <Text textStyle='quick-link-text'>
                    All stable and develop builds of Geth and tools
                  </Text>
                </ListItem>
                <ListItem>
                  <Text textStyle='quick-link-text'>
                    Archives for non-primary processor architectures
                  </Text>
                </ListItem>
                <ListItem>
                  <Text textStyle='quick-link-text'>
                    Android library archives and iOS XCode frameworks
                  </Text>
                </ListItem>
              </UnorderedList>

              <Text textStyle='quick-link-text'>
                Please select your desired platform from the lists below and download your bundle of
                choice. Please be aware that the MD5 checksums are provided by our binary hosting
                platform (Azure Blobstore) to help check for download errors. For security
                guarantees please verify any downloads via the attached PGP signature files (see{' '}
                <Link href={'#pgpsignatures'} variant='light'>
                  OpenPGP
                </Link>{' '}
                Signatures for details).
              </Text>
            </Stack>
          </SpecificVersionsSection>

          {/* STABLE RELEASES */}
          <DownloadsSection
            id='stablereleases'
            sectionDescription={
              <Text textStyle='quick-link-text'>
                These are the current and previous stable releases of go-ethereum, updated
                automatically when a new version is tagged in our{' '}
                <Link href={GETH_REPO_URL} isExternal variant='light'>
                  GitHub repository.
                </Link>
              </Text>
            }
            sectionTitle='Stable releases'
          >
            <DownloadsTable
              linuxData={ALL_LINUX_STABLE_RELEASES}
              macOSData={ALL_MACOS_STABLE_RELEASES}
              windowsData={ALL_WINDOWS_STABLE_RELEASES}
              iOSData={ALL_IOS_STABLE_RELEASES}
              androidData={ALL_ANDROID_STABLE_RELEASES}
              totalReleasesNumber={totalStableReleases}
              amountOfReleasesToShow={amountStableReleases}
              setTotalReleases={setTotalStableReleases}
            />
            <Flex
              sx={{ mt: '0 !important' }}
              flexDirection={{ base: 'column', md: 'row' }}
              justifyContent='flex-end'
              alignItems='center'
            >
              {totalStableReleases > amountStableReleases && (
                <Stack
                  sx={{ mt: '0 !important' }}
                  borderLeft={{ base: 'none', md: '2px solid var(--chakra-colors-primary)' }}
                  w={{ base: '100%', md: 'auto' }}
                >
                  <Link
                    as='button'
                    variant='button-link-secondary'
                    onClick={showMoreStableReleases}
                  >
                    <Text
                      fontFamily='"JetBrains Mono", monospace'
                      fontWeight={700}
                      textTransform='uppercase'
                      textAlign='center'
                      p={4}
                    >
                      Show older releases
                    </Text>
                  </Link>
                </Stack>
              )}
            </Flex>
          </DownloadsSection>

          {/* DEV BUILDS */}
          <DownloadsSection
            id='developbuilds'
            sectionDescription={
              <Text textStyle='quick-link-text'>
                These are the develop snapshots of go-ethereum, updated automatically when a new
                commit is pushed into our{' '}
                <Link href={GETH_REPO_URL} isExternal variant='light'>
                  GitHub repository.
                </Link>
              </Text>
            }
            sectionTitle='Develop builds'
          >
            <DownloadsTable
              linuxData={ALL_LINUX_DEV_BUILDS}
              macOSData={ALL_MACOS_DEV_BUILDS}
              windowsData={ALL_WINDOWS_DEV_BUILDS}
              iOSData={ALL_IOS_DEV_BUILDS}
              androidData={ALL_ANDROID_DEV_BUILDS}
              totalReleasesNumber={totalDevBuilds}
              amountOfReleasesToShow={amountDevBuilds}
              setTotalReleases={setTotalDevBuilds}
            />
            <Flex
              sx={{ mt: '0 !important' }}
              flexDirection={{ base: 'column', md: 'row' }}
              justifyContent='flex-end'
              alignItems='center'
            >
              {totalDevBuilds > amountDevBuilds && (
                <Stack
                  sx={{ mt: '0 !important' }}
                  borderLeft={{ base: 'none', md: '2px solid var(--chakra-colors-primary)' }}
                  w={{ base: '100%', md: 'auto' }}
                >
                  <Link as='button' variant='button-link-secondary' onClick={showMoreDevBuilds}>
                    <Text
                      fontFamily='"JetBrains Mono", monospace'
                      fontWeight={700}
                      textTransform='uppercase'
                      textAlign='center'
                      p={4}
                    >
                      Show older releases
                    </Text>
                  </Link>
                </Stack>
              )}
            </Flex>
          </DownloadsSection>

          <DownloadsSection
            id='pgpsignatures'
            sectionDescription={
              <Text textStyle='quick-link-text'>
                All the binaries available from this page are signed via our build server PGP keys:
              </Text>
            }
            sectionTitle='OpenPGP Signatures'
          >
            <Stack borderBottom='2px solid' borderColor='primary'>
              <DataTable
                columnHeaders={DOWNLOADS_OPENPGP_BUILD_HEADERS}
                data={DOWNLOADS_OPENPGP_SIGNATURES}
              />
            </Stack>

            <Stack>
              <DataTable
                columnHeaders={DOWNLOADS_OPENPGP_DEVELOPER_HEADERS}
                data={DOWNLOADS_DEVELOPERS_DATA}
              />
            </Stack>
          </DownloadsSection>

          <DownloadsSection id='importingkeys' sectionTitle='Importing keys and verifying builds'>
            <Flex
              p={4}
              borderBottom='2px'
              borderColor='primary'
              gap={4}
              flexDirection={{ base: 'column', md: 'row' }}
              alignItems='center'
            >
              <Stack flex={1}>
                <Text textStyle='quick-link-text'>
                  You can import the build server public keys by grabbing the individual keys
                  directly from the keyserver network:
                </Text>
              </Stack>

              <Stack flex={1} w={'100%'}>
                <Code p={4} bg='code-bg'>
                  gpg --recv-keys F9585DE6 C2FF8BBF 9BA28146 7B9E2481 D2A67EAC
                </Code>
              </Stack>
            </Flex>

            <Flex
              p={4}
              borderBottom='2px'
              borderColor='primary'
              gap={4}
              flexDirection={{ base: 'column', md: 'row' }}
              alignItems='center'
              sx={{ mt: '0 !important' }}
            >
              <Stack flex={1}>
                <Text textStyle='quick-link-text'>
                  Similarly you can import all the developer public keys by grabbing them directly
                  from the keyserver network:
                </Text>
              </Stack>

              <Stack flex={1} w={'100%'}>
                <Code p={4} bg='code-bg'>
                  gpg --recv-keys E058A81C 05A5DDF0 1CCB7DD2
                </Code>
              </Stack>
            </Flex>

            <Flex
              p={4}
              borderColor='primary'
              gap={4}
              flexDirection={{ base: 'column', md: 'row' }}
              alignItems='center'
              sx={{ mt: '0 !important' }}
            >
              <Stack flex={1}>
                <Text textStyle='quick-link-text'>
                  From the download listings above you should see a link both to the downloadable
                  archives as well as detached signature files. To verify the authenticity of any
                  downloaded data, grab both files and then run:
                </Text>
              </Stack>

              <Stack flex={1} w={'100%'}>
                <Code p={4} bg='code-bg'>
                  gpg --verify geth-linux-amd64-1.5.0-d0c820ac.tar.gz.asc
                </Code>
              </Stack>
            </Flex>
          </DownloadsSection>
        </Stack>
      </main>
    </>
  );
};

export default DownloadsPage;
