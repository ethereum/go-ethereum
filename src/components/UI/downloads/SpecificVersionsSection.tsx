import { Box, Flex, Grid, GridItem, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';
import { GopherHomeLinks } from '../svgs';

interface Props {
  children: React.ReactNode;
}

export const SpecificVersionsSection: FC<Props> = ({ children }) => {
  return (
    <Grid
      id='specificversions'
      templateColumns={{ base: '1fr', md: '300px 1fr' }}
      gap={{ base: 4, lg: 8 }}
    >
      <GridItem w='auto'>
        <Box h='100%'>
          {/* TODO: replace with animated/video version */}
          <Stack
            justifyContent='center'
            alignItems='center'
            p={4}
            border='2px solid'
            borderColor='primary'
            h='100%'
          >
            <GopherHomeLinks />
          </Stack>
        </Box>
      </GridItem>

      <GridItem>
        <Flex flexDir='column' border='2px solid' borderColor='primary' pb={6}>
          <Stack p={4} borderBottom='2px' borderColor='primary' sx={{ mt: '0 !important' }}>
            <Box as='h2' textStyle='h2'>
              Specific Versions
            </Box>
          </Stack>
          {children}
        </Flex>
      </GridItem>
    </Grid>
  );
};
