import { Box, Button, Flex, Heading, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { DOCS_PAGE, DOWNLOADS_PAGE } from '../../../constants';

export const HomeHero: FC = () => {
  return (
    <Stack border='2px solid' borderColor='brand.light.primary' px={4} py={{ base: 8, md: 5 }}>
      <Box mb={4}>
        <Heading
          as='h1' // TODO: move text style to theme
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={700}
          fontSize='2.75rem'
          lineHeight='3.375rem'
          letterSpacing='5%'
          color='brand.light.body'
          mb={{ base: 2, md: 4 }}
          textAlign={{ base: 'center', md: 'left' }}
        >
          go-ethereum
        </Heading>

        <Text
          // TODO: move text style to theme
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={700}
          lineHeight='21px'
          letterSpacing='0.05em'
          textAlign={{ base: 'center', md: 'left' }}
        >
          Official Go implementation of the Ethereum protocol
        </Text>
      </Box>

      <Flex
        direction={{ base: 'column', md: 'row' }}
        alignItems={{ base: 'center', md: 'flex-start' }}
      >
        <Flex direction='column' alignItems='center' mr={{ md: 6 }}>
          <NextLink href={DOWNLOADS_PAGE} passHref>
            <Button variant='primary' as='a' mb={1}>
              <Text
                // TODO: move to textstyles
                fontFamily='"JetBrains Mono", monospace'
                color='yellow.50'
                fontWeight={700}
                textTransform='uppercase'
              >
                Download
              </Text>
            </Button>
          </NextLink>

          <Text mt={1} mb={5} textStyle='hero-text-small'>
            Get our latest releases
          </Text>
        </Flex>

        <Flex direction='column' alignItems='center'>
          <NextLink href={`${DOCS_PAGE}/getting-started`} passHref>
            <Button variant='primary' as='a' mb={1}>
              <Text
                // TODO: move to textstyles
                fontFamily='"JetBrains Mono", monospace'
                color='yellow.50'
                fontWeight={700}
                textTransform='uppercase'
              >
                Documentation
              </Text>
            </Button>
          </NextLink>

          <Text mt={1} fontSize='13px' fontFamily='"Inter", sans-serif' alignSelf='center'>
            Read our documentation
          </Text>
        </Flex>
      </Flex>
    </Stack>
  );
};
