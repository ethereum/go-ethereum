import { Box, Flex, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children: React.ReactNode;
}

export const SpecificVersionsSection: FC<Props> = ({ children }) => {
  return (
    <Flex
      id='specificversions'
      border='2px solid'
      borderColor='brand.light.primary'
      flexDir={{ base: 'column', md: 'row' }}
    >
      <Flex
        p={4}
        alignItems='center'
        justifyContent='center'
        borderBottom={{ base: '2px solid #11866f', md: 'none' }}
        borderRight={{ base: 'none', md: '2px solid #11866f' }}
        flex='none'
      >
        {/* TODO: use NextImage */}
        <Image src='/images/pages/gopher-home-side-desktop.svg' alt='Gopher facing right' />
      </Flex>
      <Flex flexDir='column'>
        <Stack
          p={4}
          borderBottom='2px solid'
          borderColor='brand.light.primary'
          sx={{ mt: '0 !important' }}
        >
          <Box as='h2' textStyle='h2'>
            Specific Versions
          </Box>
        </Stack>
        {children}
      </Flex>
    </Flex>
  );
};
