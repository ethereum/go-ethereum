import { Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

export const Gopher: FC = () => {
  return (
    <Stack
      justifyContent='center'
      alignItems='center'
      p={4}
      border='2px solid'
      borderColor='primary'
      h='100%'
    >
      <Image src='/images/pages/gopher-home-side-mobile.svg' alt='Gopher greeting' />
    </Stack>
  );
};
