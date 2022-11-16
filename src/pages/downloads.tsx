import { Code, Link, ListItem, Stack, Text, UnorderedList } from '@chakra-ui/react';
import type { GetServerSideProps, NextPage } from 'next';
import { useState } from 'react';

import { DownloadsHero, DownloadsSection, DownloadsTable } from '../components/UI/downloads';
import { DataTable } from '../components/UI';

import {
  ALL_GETH_COMMITS_URL,
  DEFAULT_BUILD_AMOUNT_TO_SHOW,
  DOWNLOAD_OPENPGP_BUILD_HEADERS,
  DOWNLOAD_OPENPGP_DEVELOPER_HEADERS,
  GETH_REPO_URL,
  LATEST_GETH_RELEASE_URL,
  LATEST_SOURCES_BASE_URL,
  LINUX_BINARY_BASE_URL,
  MACOS_BINARY_BASE_URL,
  RELEASE_NOTES_BASE_URL,
  WINDOWS_BINARY_BASE_URL
} from '../constants';

import { testDownloadData } from '../data/test/download-testdata';
import { pgpBuildTestData } from '../data/test/pgpbuild-testdata';
import { pgpDeveloperTestData } from '../data/test/pgpdeveloper-testdata';

export const getServerSideProps: GetServerSideProps = async () => {
  // Latest release version number
  const versionNumber = await fetch(LATEST_GETH_RELEASE_URL)
    .then(response => response.json())
    .then(release => release.tag_name);
  // Latest release name
  const releaseName = await fetch(LATEST_GETH_RELEASE_URL)
    .then(response => response.json())
    .then(release => release.name);
  // Latest release commit hash
  const commit = await fetch(`${ALL_GETH_COMMITS_URL}/${versionNumber}`)
    .then(response => response.json())
    .then(commit => commit.sha.slice(0, 8));

  // Latest binaries urls
  const LATEST_LINUX_BINARY_URL = `${LINUX_BINARY_BASE_URL}${versionNumber.slice(
    1
  )}-${commit}.tar.gz`;
  const LATEST_MACOS_BINARY_URL = `${MACOS_BINARY_BASE_URL}${versionNumber.slice(
    1
  )}-${commit}.tar.gz`;
  const LATEST_WINDOWS_BINARY_URL = `${WINDOWS_BINARY_BASE_URL}${versionNumber.slice(
    1
  )}-${commit}.exe`;

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

  return {
    props: {
      data: { LATEST_RELEASES_DATA }
    }
  };
};

interface Props {
  data: {
    // TODO: define interfaces
    LATEST_RELEASES_DATA: {
      versionNumber: string;
      releaseName: string;
      urls: {
        LATEST_LINUX_BINARY_URL: string;
        LATEST_MACOS_BINARY_URL: string;
        LATEST_WINDOWS_BINARY_URL: string;
        LATEST_SOURCES_URL: string;
        RELEASE_NOTES_URL: string;
      };
    };
  };
}

const DownloadsPage: NextPage<Props> = ({ data }) => {
  const [amountStableReleases, updateAmountStables] = useState(DEFAULT_BUILD_AMOUNT_TO_SHOW);
  const [amountDevelopBuilds, updateAmountDevelopBuilds] = useState(DEFAULT_BUILD_AMOUNT_TO_SHOW);

  const showMoreStableReleases = () => {
    updateAmountStables(amountStableReleases + 10);
  };

  const showMoreDevelopBuilds = () => {
    updateAmountDevelopBuilds(amountDevelopBuilds + 10);
  };

  const {
    LATEST_RELEASES_DATA: { releaseName, versionNumber, urls }
  } = data;

  return (
    <>
      {/* TODO: add PageMetadata */}

      <main>
        <Stack spacing={4}>
          <DownloadsHero
            currentBuild={releaseName}
            currentBuildVersion={versionNumber}
            linuxBuildURL={urls.LATEST_LINUX_BINARY_URL}
            macOSBuildURL={urls.LATEST_MACOS_BINARY_URL}
            releaseNotesURL={urls.RELEASE_NOTES_URL}
            sourceCodeURL={urls.LATEST_SOURCES_URL}
            windowsBuildURL={urls.LATEST_WINDOWS_BINARY_URL}
          />

          <DownloadsSection
            imgSrc='/images/pages/gopher-home-side-desktop.svg'
            imgAltText='Gopher facing right'
            sectionTitle='Specific Versions'
            id='specificversions'
          >
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
          </DownloadsSection>

          <DownloadsSection sectionTitle='Stable releases' id='stablereleases'>
            <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
              <Text textStyle='quick-link-text'>
                These are the current and previous stable releases of go-ethereum, updated
                automatically when a new version is tagged in our{' '}
                <Link href={GETH_REPO_URL} isExternal variant='light'>
                  GitHub repository.
                </Link>
              </Text>
            </Stack>

            {/* TODO: swap test data for real data */}
            <DownloadsTable data={testDownloadData.slice(0, amountStableReleases)} />

            <Stack sx={{ mt: '0 !important' }}>
              <Link as='button' variant='button-link-secondary' onClick={showMoreStableReleases}>
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
          </DownloadsSection>

          <DownloadsSection sectionTitle='Develop builds' id='developbuilds'>
            <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
              <Text textStyle='quick-link-text'>
                These are the develop snapshots of go-ethereum, updated automatically when a new
                commit is pushed into our{' '}
                <Link href={GETH_REPO_URL} isExternal variant='light'>
                  GitHub repository.
                </Link>
              </Text>
            </Stack>

            {/* TODO: swap for real data */}
            <DownloadsTable data={testDownloadData.slice(0, amountDevelopBuilds)} />

            <Stack sx={{ mt: '0 !important' }}>
              <Link as='button' variant='button-link-secondary' onClick={showMoreDevelopBuilds}>
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
          </DownloadsSection>

          <DownloadsSection sectionTitle='OpenPGP Signatures' id='pgpsignatures'>
            <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
              <Text textStyle='quick-link-text'>
                All the binaries available from this page are signed via our build server PGP keys:
              </Text>
            </Stack>

            {/* TODO: swap for real data */}
            <Stack borderBottom='2px solid' borderColor='brand.light.primary'>
              <DataTable columnHeaders={DOWNLOAD_OPENPGP_BUILD_HEADERS} data={pgpBuildTestData} />
            </Stack>

            {/* TODO: swap for real data */}
            <Stack>
              <DataTable
                columnHeaders={DOWNLOAD_OPENPGP_DEVELOPER_HEADERS}
                data={pgpDeveloperTestData}
              />
            </Stack>
          </DownloadsSection>

          <DownloadsSection sectionTitle='Importing keys and verifying builds' id='importingkeys'>
            <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
              <Text textStyle='quick-link-text'>
                You can import the build server public keys by grabbing the individual keys directly
                from the keyserver network:
              </Text>

              {/* TODO: These keys depends on the binary */}
              <Code p={4}>gpg --recv-keys F9585DE6 C2FF8BBF 9BA28146 7B9E2481 D2A67EAC</Code>
            </Stack>

            <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
              <Text textStyle='quick-link-text'>
                Similarly you can import all the developer public keys by grabbing them directly
                from the keyserver network:
              </Text>

              {/* TODO: These are developer keys, do we need to change? */}
              <Code p={4}>gpg --recv-keys E058A81C 05A5DDF0 1CCB7DD2</Code>
            </Stack>

            <Stack p={4}>
              <Text textStyle='quick-link-text'>
                From the download listings above you should see a link both to the downloadable
                archives as well as detached signature files. To verify the authenticity of any
                downloaded data, grab both files and then run:
              </Text>

              {/* TODO: These keys depends on the binary */}
              <Code p={4}>gpg --verify geth-linux-amd64-1.5.0-d0c820ac.tar.gz.asc</Code>
            </Stack>
          </DownloadsSection>
        </Stack>
      </main>
    </>
  );
};

export default DownloadsPage;
