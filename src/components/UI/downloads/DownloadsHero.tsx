import { Box, Button, Center, Grid, HStack, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { GopherDownloads } from '../svgs';

import { DOWNLOAD_HEADER_BUTTONS } from '../../../constants';

interface DownloadsHero {
  currentBuild: string;
  currentBuildVersion: string;
  linuxBuildURL: string;
  macOSBuildURL: string;
  releaseNotesURL: string;
  sourceCodeURL: string;
  windowsBuildURL: string;
}

export const DownloadsHero: FC<DownloadsHero> = ({
  currentBuild,
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
    <Grid
      border='2px solid'
      borderColor='primary'
      p={4}
      templateColumns={{ base: 'repeat(1, 1fr)', lg: '1fr 430px' }}
      gap={4}
    >
      <Stack>
        <Grid
          mb={4}
          templateColumns={{ base: 'repeat(1, 1fr)', md: '1fr 300px', lg: '1fr' }}
          gap={4}
          py={4}
        >
          <Stack>
            <Box as='h1' textStyle='h1'>
              Download go-ethereum
            </Box>
            <Text
              // TODO: move text style to theme
              fontFamily='"JetBrains Mono", monospace'
              lineHeight='21px'
              mb={{ base: '4 !important', md: '8 !important' }}
              color='body'
            >
              {currentBuild}
            </Text>

            <Text mb={4} color='body'>
              You can download the latest 64-bit stable release of Geth for our primary platforms
              below. Packages for all supported platforms, as well as develop builds, can be found
              further down the page. If you&apos;re looking to install Geth and/or associated tools
              via your favorite package manager, please check our installation guide.
            </Text>
          </Stack>

          <Center
            p={{ base: 0, md: 8 }}
            flex={{ base: 'none' }}
            display={{ base: 'block', lg: 'none' }}
            order={{ base: -1, md: 1 }}
          >
            <GopherDownloads aria-label='Gopher plugged in' w={{ base: '100%' }} />
          </Center>
        </Grid>

        <Grid templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)' }} gap={4}>
          {Object.keys(DOWNLOAD_HEADER_BUTTONS).map((key: string) => {
            const { name, buildURL, Svg, ariaLabel } = DOWNLOAD_HEADER_BUTTONS[key];

            return (
              <NextLink key={key} href={buildURL} passHref legacyBehavior>
                <Button as='a' variant='primary' width={{ base: '100%' }} h={16} data-group>
                  <HStack spacing={4}>
                    <Stack alignItems='center'>
                      <Svg
                        aria-label={ariaLabel}
                        maxH='44px'
                        _groupHover={{ color: 'yellow.50' }}
                        _groupFocus={{ color: 'yellow.50' }}
                        _groupActive={{ color: 'yellow.50' }}
                      />
                    </Stack>
                    <Box>
                      <Text textStyle='downloads-button-label'>For {name}</Text>
                      <Text textStyle='downloads-button-sublabel'>geth {currentBuildVersion}</Text>
                    </Box>
                  </HStack>
                </Button>
              </NextLink>
            );
          })}
        </Grid>

        <Box textAlign={'center'} pt={1} pb={2}>
          <Link href={releaseNotesURL} isExternal variant='light'>
            Release notes for {currentBuild}
          </Link>
        </Box>
      </Stack>

      <Center display={{ base: 'none', lg: 'flex' }}>
        <GopherDownloads aria-label='Gopher plugged in' w={96} />
      </Center>
    </Grid>
  );
};
