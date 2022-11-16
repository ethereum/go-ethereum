import { Box, Grid, GridItem, Image, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

import { GopherHomeNodes } from '../svgs/GopherHomeNodes';
import { ETHEREUM_ORG_RUN_A_NODE_URL } from '../../../constants';

interface Props {
  children: React.ReactNode;
}

export const WhyRunANode: FC<Props> = ({ children }) => {
  return (
    <Stack border='2px solid' borderColor='primary'>
      <Grid
        templateColumns={{ base: 'repeat(1, 1fr)', md: 'repeat(2, 1fr)' }}
        borderBottom={{ base: 'none', md: '2px' }}
        borderColor='primary !important'
      >
        <GridItem order={{ base: 1, md: 2 }}>
          <Stack
            p={4}
            borderBottom='2px'
            borderColor='primary'
            sx={{ mt: '0 !important' }}
          >
            <Box as='h2' textStyle='h2'>
              Why run a node?
            </Box>
          </Stack>

          <Stack
            p={4}
            borderBottom={{ base: '2px', md: 'none' }}
            borderColor='primary'
            sx={{ mt: '0 !important' }}
          >
            {children}
          </Stack>
        </GridItem>

        <GridItem rowSpan={2}>
          <Stack
            justifyContent='center'
            alignItems='center'
            p={4}
            borderBottom={{ base: '2px', md: 'none' }}
            borderRight={{ base: 'none', md: '2px' }}
            borderColor='primary !important'
            h='100%'
          >
            <GopherHomeNodes />
          </Stack>
        </GridItem>
      </Grid>

      <Stack sx={{ mt: '0 !important' }}>
        <NextLink href={ETHEREUM_ORG_RUN_A_NODE_URL} passHref>
          <Link variant='button-link-secondary' isExternal>
            <Text textStyle='home-section-link-label'>Read more about running a node</Text>
          </Link>
        </NextLink>
      </Stack>
    </Stack>
  );
};
