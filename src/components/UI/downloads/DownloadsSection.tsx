import { Box, IconProps, Stack } from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  id: string;
  sectionTitle: string;
  children: React.ReactNode;
  Svg?: React.FC<IconProps>;
  ariaLabel?: string;
}

export const DownloadsSection: FC<Props> = ({ children, Svg, ariaLabel, sectionTitle, id, showGopher }) => {
  return (
    <Stack border='2px solid' borderColor='primary' id={id}>
      {Svg && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='primary'>
          <Svg aria-label={ariaLabel} />
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
