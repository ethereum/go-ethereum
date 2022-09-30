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
          {/* TODO: define button variants */}
          <NextLink href={DOWNLOADS_PAGE} passHref>
            {/* TODO: update text */}
            <Button
              as='a'
              py='8px'
              px='32px'
              borderRadius={0}
              width={{ base: '188px', md: 'auto' }}
              // TODO: move to theme colors
              bg='brand.light.primary'
              _hover={{ bg: 'brand.light.secondary' }}
              _focus={{
                bg: 'brand.light.primary',
                boxShadow: 'inset 0 0 0 2px #06fece !important'
              }}
              _active={{ borderTop: '4px solid', borderColor: 'green.200', pt: '4px' }}
              mb={1}
            >
              <Text
                fontFamily='"JetBrains Mono", monospace'
                // TODO: move to theme colors
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
          {/* TODO: define button variants */}
          <NextLink href={`${DOCS_PAGE}/getting-started`} passHref>
            <Button
              as='a'
              py='8px'
              px='32px'
              borderRadius={0}
              bg='#11866F'
              _hover={{ bg: '#25453f' }}
              _focus={{ bg: '#11866f', boxShadow: 'inset 0 0 0 2px #06fece !important' }}
              _active={{ borderTop: '4px solid #06fece', pt: '4px' }}
              mb={1}
            >
              <Text
                fontFamily='"JetBrains Mono", monospace'
                // TODO: move to theme colors
                color='#F0F2E2'
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
