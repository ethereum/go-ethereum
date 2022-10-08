import { Box, Grid, GridItem, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { CONTRIBUTING_PAGE, DOCS_PAGE, FAQ_PAGE } from '../../../constants';

export const QuickLinks: FC = () => {
  return (
    <Stack border='2px solid' borderColor='brand.light.primary'>
      <Stack p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
        <Box as='h2' textStyle='h2'>
          Quick Links
        </Box>
      </Stack>

      <Grid templateColumns='repeat(2, 1fr)' sx={{ mt: '0 !important' }}>
        {/* get started */}
        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
        >
          <Stack p={4} h='100%'>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px' mb={-1}>
              Don&apos;t know where to start?
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              We can help.
            </Text>
          </Stack>
        </GridItem>
        <GridItem borderBottom='2px solid' borderColor='brand.light.primary'>
          <NextLink href={`${DOCS_PAGE}/getting-started`} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='green.50'
                _hover={{ textDecoration: 'none', bg: 'brand.light.primary', color: 'yellow.50' }}
                _focus={{
                  textDecoration: 'none',
                  bg: 'brand.light.primary',
                  color: 'yellow.50',
                  boxShadow: 'inset 0 0 0 3px #f0f2e2 !important'
                }}
                _active={{
                  textDecoration: 'none',
                  bg: 'brand.light.secondary',
                  color: 'yellow.50'
                }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  fontWeight={700}
                  textTransform='uppercase'
                  textAlign='center'
                  color='brand.light.primary'
                  _groupHover={{ color: 'yellow.50' }}
                  _groupActive={{ color: 'yellow.50' }}
                  _groupFocus={{ color: 'yellow.50' }}
                >
                  Get started
                </Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>

        {/* faq */}
        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
        >
          <Stack p={4} h='100%'>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px' mb={-1}>
              Have doubts?
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Check the FAQ section in the documentation.
            </Text>
          </Stack>
        </GridItem>
        <GridItem borderBottom='2px solid' borderColor='brand.light.primary'>
          <NextLink href={FAQ_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='green.50'
                _hover={{ textDecoration: 'none', bg: 'brand.light.primary', color: 'yellow.50' }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to theme colors
                  fontWeight={700}
                  textTransform='uppercase'
                  textAlign='center'
                  color='brand.light.primary'
                  _groupHover={{ color: 'yellow.50' }}
                >
                  Go to the FAQ
                </Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>

        {/* how to contribute */}
        <GridItem borderRight='2px solid' borderColor='brand.light.primary'>
          <Stack p={4} h='100%'>
            <Text fontFamily='"Inter", sans-serif' lineHeight='26px' mb={-1}>
              Want to know how to contribute?
            </Text>

            <Text fontFamily='"Inter", sans-serif' lineHeight='26px'>
              Get more information in the documentation.
            </Text>
          </Stack>
        </GridItem>
        <GridItem>
          <NextLink href={CONTRIBUTING_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='green.50'
                _hover={{ textDecoration: 'none', bg: 'brand.light.primary', color: 'yellow.50' }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text
                  fontFamily='"JetBrains Mono", monospace'
                  // TODO: move to text style
                  fontWeight={700}
                  textTransform='uppercase'
                  textAlign='center'
                  color='brand.light.primary'
                  _groupHover={{ color: 'yellow.50' }}
                >
                  How to contribute
                </Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>
      </Grid>
    </Stack>
  );
};
