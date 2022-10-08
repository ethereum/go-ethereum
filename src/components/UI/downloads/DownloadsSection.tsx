import { Box, Image, Stack } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  children?: React.ReactNode;
  imgSrc?: string;
  imgAltText?: string;
  sectionTitle: string
}

export const DownloadsSection: FC<Props> = ({
  children,
  imgSrc,
  imgAltText,
  sectionTitle,
}) => {
  return (
    <Stack border='2px solid' borderColor='brand.light.primary'>
      {!!imgSrc && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='brand.light.primary'>
          {/* TODO: use NextImage */}
          <Image src={imgSrc} alt={imgAltText} />
        </Stack>
      )}

      <Stack
        p={4}
        borderBottom='2px solid'
        borderColor='brand.light.primary'
        sx={{ mt: '0 !important' }}
      >
        <Box as='h2' textStyle='h2'>
          {sectionTitle}
        </Box>
      </Stack>

      <Stack spacing={4}>
        {children}
      </Stack>
    </Stack>
  )
}