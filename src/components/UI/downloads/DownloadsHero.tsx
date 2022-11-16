import { Box, Button, Image, Link, Stack, HStack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { DOWNLOAD_HEADER_BUTTONS } from '../../../constants';

interface DownloadsHero {
  currentBuildName: string;
  currentBuildVersion: string;
  linuxBuildURL: string;
  macOSBuildURL: string;
  releaseNotesURL: string;
  sourceCodeURL: string;
  windowsBuildURL: string;
}

export const DownloadsHero: FC<DownloadsHero> = ({
  currentBuildName,
  currentBuildVersion,
  linuxBuildURL,
  macOSBuildURL,
  releaseNotesURL,
  sourceCodeURL,
  windowsBuildURL
}) => {
  DOWNLOAD_HEADER_BUTTONS.linuxBuild.buildURL = linuxBuildURL;
  DOWNLOAD_HEADER_BUTTONS.macOSBuild.buildURL = macOSBuildURL;
  DOWNLOAD_HEADER_BUTTONS.windowsBuild.buildURL = windowsBuildURL;
  DOWNLOAD_HEADER_BUTTONS.sourceCode.buildURL = sourceCodeURL;

  return (
    <Stack border='3px solid' borderColor='primary' py={4} px={4}>
      <Stack alignItems='center'>
        <Image src='/images/pages/gopher-downloads-front-light.svg' alt='Gopher plugged in' />
      </Stack>

      <Box mb={4}>
        <Box as='h1' textStyle='h1'>
          Download go-ethereum
        </Box>

        <Text
          // TODO: move text style to theme
          fontFamily='"JetBrains Mono", monospace'
          lineHeight='21px'
          mb={8}
        >
          {currentBuildName} ({currentBuildVersion})
        </Text>

        <Text mb={4}>
          You can download the latest 64-bit stable release of Geth for our primary platforms below.
          Packages for all supported platforms, as well as develop builds, can be found further down
          the page. If you&apos;re looking to install Geth and/or associated tools via your favorite
          package manager, please check our installation guide.
        </Text>

        {Object.keys(DOWNLOAD_HEADER_BUTTONS).map((key: string) => {
          return (
            <NextLink key={key} href={DOWNLOAD_HEADER_BUTTONS[key].buildURL} passHref>
              <Button as='a' variant='primary' width={{ base: '100%' }} p={8} mb={4}>
                <HStack spacing={4}>
                  <Stack alignItems='center'>
                    <Image
                      src={DOWNLOAD_HEADER_BUTTONS[key].image}
                      alt={DOWNLOAD_HEADER_BUTTONS[key].imageAlt}
                    />
                  </Stack>
                  <Box>
                    <Text textStyle='downloads-button-label'>
                      For {DOWNLOAD_HEADER_BUTTONS[key].name}
                    </Text>
                    <Text textStyle='downloads-button-label'>geth {currentBuildName}</Text>
                  </Box>
                </HStack>
              </Button>
            </NextLink>
          );
        })}

        <Box textAlign={'center'}>
          <Link href={releaseNotesURL} isExternal variant='light'>
            Release notes for {currentBuildName} {currentBuildVersion}
          </Link>
        </Box>
      </Box>
    </Stack>
  );
};
