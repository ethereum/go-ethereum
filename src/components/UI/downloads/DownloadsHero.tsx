import { Box, Button, Heading, Image, Link, Stack, HStack, Text } from '@chakra-ui/react';
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
        <Heading
          as='h1' // TODO: move text style to theme
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={700}
          fontSize='2.75rem'
          lineHeight='3.375rem'
          letterSpacing='5%'
          color='brand.light.body'
        >
          Download go-ethereum
        </Heading>

        <Text
          // TODO: move text style to theme
          fontFamily='"JetBrains Mono", monospace'
          lineHeight='21px'
          mb={8}
        >
          {currentBuildName} ({currentBuildVersion})
        </Text>

        <Text
          mb={4}
        >
          You can download the latest 64-bit stable release of Geth for our primary platforms below. Packages for all supported platforms, as well as develop builds, can be found further down the page. If you&apos;re looking to install Geth and/or associated tools via your favorite package manager, please check our installation guide.
        </Text>

        <NextLink href={linuxBuildURL} passHref>
          <Button
            as='a'
            p={8}
            borderRadius={0}
            width={{ base: '100%' }}
            // TODO: move to theme colors
            bg='brand.light.primary'
            _hover={{ bg: 'brand.light.secondary' }}
            _focus={{
              bg: 'brand.light.primary',
              boxShadow: 'inset 0 0 0 2px #06fece !important'
            }}
            _active={{ borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/linux-penguin.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontWeight={700}
                  textTransform='uppercase'
                >
                  For linux
                </Text>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontSize='xs'
                  textTransform='uppercase'
                >
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={macOSBuildURL} passHref>
          <Button
            as='a'
            p={8}
            borderRadius={0}
            width={{ base: '100%' }}
            // TODO: move to theme colors
            bg='brand.light.primary'
            _hover={{ bg: 'brand.light.secondary' }}
            _focus={{
              bg: 'brand.light.primary',
              boxShadow: 'inset 0 0 0 2px #06fece !important'
            }}
            _active={{ borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/macos-logo.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontWeight={700}
                  textTransform='uppercase'
                >
                  For macos
                </Text>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontSize='xs'
                  textTransform='uppercase'
                >
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={windowsBuildURL} passHref>
          <Button
            as='a'
            p={8}
            borderRadius={0}
            width={{ base: '100%' }}
            // TODO: move to theme colors
            bg='brand.light.primary'
            _hover={{ bg: 'brand.light.secondary' }}
            _focus={{
              bg: 'brand.light.primary',
              boxShadow: 'inset 0 0 0 2px #06fece !important'
            }}
            _active={{ borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/windows-logo.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontWeight={700}
                  textTransform='uppercase'
                >
                  For windows
                </Text>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontSize='xs'
                  textTransform='uppercase'
                >
                  geth {currentBuildName}
                </Text>
              </Box>
            </HStack>
          </Button>
        </NextLink>

        <NextLink href={sourceCodeURL} passHref>
          <Button
            as='a'
            p={8}
            borderRadius={0}
            width={{ base: '100%' }}
            // TODO: move to theme colors
            bg='brand.light.primary'
            _hover={{ bg: 'brand.light.secondary' }}
            _focus={{
              bg: 'brand.light.primary',
              boxShadow: 'inset 0 0 0 2px #06fece !important'
            }}
            _active={{ borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }}
            mb={4}
          >
            <HStack spacing={4}>
              <Box>
                <Image m={'auto'} src='/images/pages/source-branch.svg' alt='Gopher greeting' />
              </Box>  
              <Box>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontWeight={700}
                  textTransform='uppercase'
                >
                  Sources
                </Text>
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  color='yellow.50'
                  fontSize='xs'
                  textTransform='uppercase'
                >
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
            color='#11866f'
            _hover={{ color: '#1d242c', textDecorationColor: '#1d242c' }}
            _focus={{
              color: '#11866f',
              boxShadow: '0 0 0 1px #11866f !important',
              textDecoration: 'none'
            }}
            _pressed={{ color: '#25453f', textDecorationColor: '#25453f' }}
          >
            Release notes for {currentBuildName} {currentBuildVersion}
          </Link>
        </Box>
      </Box>
    </Stack>
  );
};
