import { Grid, GridItem, Image, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { DOCS_PAGE, DOWNLOADS_PAGE, GETH_TWITTER_URL } from '../../constants'

export const Footer: FC = () => {
  return (
    <Stack mt={4} border='2px solid' borderColor='brand.light.primary'>
      <Grid templateColumns='repeat(2, 1fr)' sx={{ mt: '0 !important' }}>
        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
        >
          <Stack
            p={4}
            _hover={{
              textDecoration: 'none',
              bg: 'brand.light.primary',
              color: 'yellow.50 !important'
            }}
          >
            <NextLink href={DOWNLOADS_PAGE} passHref>
              <Link _hover={{ textDecoration: 'none' }}>
                <Text textStyle='quick-link-label'>DOWNLOADS</Text>
              </Link>
            </NextLink>
          </Stack>
        </GridItem>

        <GridItem
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          p={4}
        >
          <NextLink href={DOCS_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Text textStyle='quick-link-label'>DOCUMENTATION</Text>
            </Link>
          </NextLink>
        </GridItem>
      </Grid>

      <Grid templateColumns='repeat(3, 1fr)' sx={{ mt: '0 !important' }}>
        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          alignItems="center"
        >
          <Stack alignItems='center' p={4}>
            <NextLink href={GETH_TWITTER_URL} passHref>
              <Link isExternal>
                <Image src="/images/pages/twitter.svg" alt="Twitter logo" />
              </Link>
            </NextLink>
          </Stack>
        </GridItem>

        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          alignItems="center"
        >
          <Stack alignItems='center' p={4}>
            <Image src="/images/pages/discord.svg" alt="Discord logo" />
          </Stack>
        </GridItem>

        <GridItem
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          alignItems="center"
        >
          <Stack alignItems='center' p={4}>
            <Image src="/images/pages/github.svg" alt="GitHub logo" />
          </Stack>
        </GridItem>
      </Grid>

      <Stack p={4} sx={{ mt: '0 !important' }} textAlign='center'>
        <Text textStyle='footer-text'>© 2013–2022. The go-ethereum Authors.</Text>
      </Stack>
    </Stack>
  )
}