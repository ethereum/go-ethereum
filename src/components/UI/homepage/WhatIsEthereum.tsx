import { Box, Grid, GridItem, Stack } from '@chakra-ui/react';
import { FC } from 'react';
import { GlyphHome } from '../svgs/GlyphHome';
import { ETHEREUM_ORG_URL } from '../../../constants';
import { ButtonLinkSecondary } from '..';

interface Props {
  children: React.ReactNode;
}

export const WhatIsEthereum: FC<Props> = ({ children }) => {
  return (
    <Stack border='2px' borderColor='primary'>
      <Grid
        templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)' }}
        borderBottom='2px'
        borderColor='primary'
      >
        <GridItem
          order={{ base: 2, md: 1 }}
          borderRight={{ base: 'none', md: '2px' }}
          borderColor='primary !important'
        >
          <Stack p={4} borderBottom='2px' borderColor='primary' sx={{ mt: '0 !important' }}>
            <Box as='h2' textStyle='h2'>
              What is Ethereum?
            </Box>
          </Stack>

          <Stack p={4} sx={{ mt: '0 !important' }}>
            {children}
          </Stack>
        </GridItem>

        <GridItem order={{ base: 1, md: 2 }}>
          <Stack
            justifyContent='center'
            alignItems='center'
            p={4}
            borderBottom={{ base: '2px', md: 'none' }}
            borderColor='primary'
            h='100%'
          >
            <GlyphHome />
          </Stack>
        </GridItem>
      </Grid>

      <ButtonLinkSecondary href={ETHEREUM_ORG_URL}>Learn more on Ethereum.org</ButtonLinkSecondary>
    </Stack>
  );
};
