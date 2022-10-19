import { Grid, GridItem, Image, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import {
  DOCS_PAGE,
  DOWNLOADS_PAGE,
  GETH_DISCORD_URL,
  GETH_REPO_URL,
  GETH_TWITTER_URL
} from '../../constants'

export const Footer: FC = () => {
  return (
    <Stack mt={4} border='2px solid' borderColor='brand.light.primary'>
      <Grid templateColumns='repeat(2, 1fr)' sx={{ mt: '-2px !important' }}>
        <GridItem
          borderRight='2px solid'
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          color='brand.light.primary'
            _hover={{
              textDecoration: 'none',
              bg: 'brand.light.primary',
              color: 'yellow.50 !important'
            }}
        >
          <NextLink href={DOWNLOADS_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Text textStyle='home-section-link-label'>DOWNLOADS</Text>
            </Link>
          </NextLink>
        </GridItem>

        <GridItem
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          color='brand.light.primary'
            _hover={{
              textDecoration: 'none',
              bg: 'brand.light.primary',
              color: 'yellow.50 !important'
            }}
        >
          <NextLink href={DOCS_PAGE} passHref>
            <Link _hover={{ textDecoration: 'none' }}>
              <Text textStyle='home-section-link-label'>DOCUMENTATION</Text>
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
          _hover={{
            textDecoration: 'none',
            bg: 'brand.light.primary',
            color: 'yellow.50 !important'
          }}
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
          _hover={{
            textDecoration: 'none',
            bg: 'brand.light.primary',
            color: 'yellow.50 !important'
          }}
        >
          <Stack alignItems='center' p={4}>
            <NextLink href={GETH_DISCORD_URL} passHref>
              <Link isExternal>
                <Image src="/images/pages/discord.svg" alt="Discord logo" />
              </Link>
            </NextLink>
          </Stack>
        </GridItem>

        <GridItem
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          alignItems="center"
          _hover={{
            textDecoration: 'none',
            bg: 'brand.light.primary',
            color: 'yellow.50 !important'
          }}
        >
          <Stack alignItems='center' p={4}>
            <NextLink href={GETH_REPO_URL} passHref>
              <Link isExternal>
                <Image src="/images/pages/github.svg" alt="GitHub logo" />
              </Link>
            </NextLink>
          </Stack>
        </GridItem>
      </Grid>

      <Stack p={4} sx={{ mt: '0 !important' }} textAlign='center'>
        <Text textStyle='footer-text'>© 2013–2022. The go-ethereum Authors.</Text>
      </Stack>
    </Stack>
  )
}