import { Box, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

import { GopherHomeLinks } from '../svgs'
interface Props {
  children: React.ReactNode;
  id: string;
  imgSrc?: string;
  imgAltText?: string;
  sectionTitle: string;
  showGopher?: boolean;
}

export const DownloadsSection: FC<Props> = ({ children, imgSrc, imgAltText, sectionTitle, id, showGopher }) => {
  return (
    <Stack border='2px solid' borderColor='primary' id={id}>
      {imgSrc || showGopher && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='primary'>
          {/* TODO: use NextImage */}
          {imgSrc && <Image src={imgSrc} alt={imgAltText} />}
          {showGopher && < GopherHomeLinks/>}
        </Stack>
      )}

      <Stack
        p={4}
        borderBottom='2px solid'
        borderColor='primary'
        sx={{ mt: '0 !important' }}
      >
        <Box as='h2' textStyle='h2'>
          {sectionTitle}
        </Box>
      </Stack>

      <Stack spacing={4}>{children}</Stack>
    </Stack>
  );
};
