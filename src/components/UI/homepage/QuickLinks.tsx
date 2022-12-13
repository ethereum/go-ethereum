import { Box, Grid, GridItem, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { CONTRIBUTING_PAGE, DOCS_PAGE, FAQ_PAGE } from '../../../constants';

export const QuickLinks: FC = () => {
  return (
    <Stack border='2px solid' borderColor='primary'>
      <Stack p={4} borderBottom='2px solid' borderColor='primary'>
        <Box as='h2' textStyle='h2'>
          Quick Links
        </Box>
      </Stack>

      <Grid
        templateColumns={{ base: 'repeat(2, 1fr)', md: '1fr auto' }}
        sx={{ mt: '0 !important' }}
      >
        {/* get started */}
        <GridItem borderRight='2px solid' borderBottom='2px solid' borderColor='primary'>
          <Stack p={4} h='100%'>
            <Text textStyle='quick-link-text' mb={-1}>
              Don&apos;t know where to start?
            </Text>

            <Text textStyle='quick-link-text'>We can help.</Text>
          </Stack>
        </GridItem>
        <GridItem borderBottom='2px solid' borderColor='primary'>
          <NextLink href={`${DOCS_PAGE}/getting-started`} passHref legacyBehavior>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='button-bg'
                _hover={{ textDecoration: 'none', bg: 'primary', color: 'bg' }}
                _focus={{
                  textDecoration: 'none',
                  bg: 'primary',
                  color: 'bg',
                  boxShadow: 'inset 0 0 0 3px #f0f2e2 !important'
                }}
                _active={{
                  textDecoration: 'none',
                  bg: 'secondary',
                  color: 'bg'
                }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text textStyle='quick-link-label'>Get started</Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>

        {/* faq */}
        <GridItem borderRight='2px solid' borderBottom='2px solid' borderColor='primary'>
          <Stack p={4} h='100%'>
            <Text textStyle='quick-link-text' mb={-1}>
              Have doubts?
            </Text>

            <Text textStyle='quick-link-text'>Check the FAQ section in the documentation.</Text>
          </Stack>
        </GridItem>
        <GridItem borderBottom='2px solid' borderColor='primary'>
          <NextLink href={FAQ_PAGE} passHref legacyBehavior>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='button-bg'
                _hover={{ textDecoration: 'none', bg: 'primary', color: 'bg' }}
                _focus={{
                  textDecoration: 'none',
                  bg: 'primary',
                  color: 'bg',
                  boxShadow: 'inset 0 0 0 3px var(--chakra-colors-bg) !important'
                }}
                _active={{
                  textDecoration: 'none',
                  bg: 'secondary',
                  color: 'bg'
                }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text textStyle='quick-link-label'>Go to the FAQ</Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>

        {/* how to contribute */}
        <GridItem borderRight='2px solid' borderColor='primary'>
          <Stack p={4} h='100%'>
            <Text textStyle='quick-link-text' mb={-1}>
              Want to know how to contribute?
            </Text>

            <Text textStyle='quick-link-text'>Get more information in the documentation.</Text>
          </Stack>
        </GridItem>
        <GridItem>
          <NextLink href={CONTRIBUTING_PAGE} passHref legacyBehavior>
            <Link _hover={{ textDecoration: 'none' }}>
              <Stack
                data-group
                bg='button-bg'
                _hover={{ textDecoration: 'none', bg: 'primary', color: 'bg' }}
                _focus={{
                  textDecoration: 'none',
                  bg: 'primary',
                  color: 'bg',
                  boxShadow: 'inset 0 0 0 3px var(--chakra-colors-bg) !important'
                }}
                _active={{
                  textDecoration: 'none',
                  bg: 'secondary',
                  color: 'bg'
                }}
                justifyContent='center'
                h='100%'
                p={4}
              >
                <Text textStyle='quick-link-label'>How to contribute</Text>
              </Stack>
            </Link>
          </NextLink>
        </GridItem>
      </Grid>
    </Stack>
  );
};
