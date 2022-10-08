import { Box, Button, Image, Link, Stack, HStack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

interface DownloadsHero {
  currentBuildName: string
  currentBuildVersion: string
  linuxBuildURL: string
  macOSBuildURL: string
  releaseNotesURL: string
  sourceCodeURL: string
  windowsBuildURL: string
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
  return (
    <Stack border='3px solid' borderColor='brand.light.primary' py={4} px={4}>
      <Box>
        <Image w={'180px'} m={'auto'} src='/images/pages/gopher-downloads-front-light.svg' alt='Gopher greeting' />  
      </Box>

      <Box mb={4}>
        <Box
          as='h1'
          textStyle='h1'
        >
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
          You can download the latest 64-bit stable release of Geth for our primary platforms below. Packages for all supported platforms, as well as develop builds, can be found further down the page. If you&apos;re looking to install Geth and/or associated tools via your favorite package manager, please check our installation guide.
        </Text>

        <NextLink href={linuxBuildURL} passHref>
          <Button
            as='a'
            variant='primary'
            width={{ base: '100%' }}
            p={8}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/linux-penguin.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text textStyle='downloads-button-label'>
                  For linux
                </Text>
                <Text textStyle='downloads-button-label'>
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={macOSBuildURL} passHref>
          <Button
            as='a'
            variant='primary'
            width={{ base: '100%' }}
            p={8}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/macos-logo.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text textStyle='downloads-button-label'>
                  For macos
                </Text>
                <Text textStyle='downloads-button-label'>
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={windowsBuildURL} passHref>
          <Button
            as='a'
            variant='primary'
            width={{ base: '100%' }}
            p={8}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/windows-logo.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text textStyle='downloads-button-label'>
                  For windows
                </Text>
                <Text textStyle='downloads-button-label'>
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={sourceCodeURL} passHref>
          <Button
            as='a'
            variant='primary'
            width={{ base: '100%' }}
            p={8}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/source-branch.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text textStyle='downloads-button-label'>
                  Sources
                </Text>
                <Text textStyle='downloads-button-label'>
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <Box textAlign={'center'}>
          <Link
            href={releaseNotesURL}
            isExternal
            color='brand.light.primary'
            _hover={{ color: 'brand.light.body', textDecorationColor: 'brand.light.body' }}
            _focus={{
              color: 'brand.light.primary',
              boxShadow: 'linkBoxShadow',
              textDecoration: 'none'
            }}
            _pressed={{ color: 'brand.light.secondary', textDecorationColor: 'brand.light.secondary' }}
          >
            Release notes for {currentBuildName} {currentBuildVersion}
          </Link>
        </Box>
      </Box>
    </Stack>
  );
};
