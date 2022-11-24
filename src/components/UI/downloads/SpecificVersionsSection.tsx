import { Box, Flex, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';
import { GopherHomeLinks } from '../svgs';

interface Props {
  children: React.ReactNode;
}

export const SpecificVersionsSection: FC<Props> = ({ children }) => {
  return (
    <Flex
      id='specificversions'
      border='2px'
      borderColor='primary'
      flexDir={{ base: 'column', md: 'row' }}
    >
      <Flex
        p={4}
        alignItems='center'
        justifyContent='center'
        borderBottom={{ base: '2px', md: 'none' }}
        borderRight={{ base: 'none', md: '2px' }}
        borderColor='primary !important'
        flex='none'
      >
        <GopherHomeLinks />
      </Flex>
      <Flex flexDir='column'>
        <Stack p={4} borderBottom='2px' borderColor='primary' sx={{ mt: '0 !important' }}>
          <Box as='h2' textStyle='h2'>
            Specific Versions
          </Box>
        </Stack>
        {children}
      </Flex>
    </Flex>
  );
};
