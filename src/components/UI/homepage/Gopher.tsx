import { Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

export const Gopher: FC = () => {
  return (
    <Stack alignItems='center' p={4} border='2px solid' borderColor='brand.light.primary'>
      <Image src='/images/pages/gopher-home-side-mobile.svg' alt='Gopher greeting' />
    </Stack>
  );
};
