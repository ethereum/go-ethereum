import { Box, IconProps, Stack } from '@chakra-ui/react';
import { FC } from 'react';

import { ButtonLinkSecondary } from '..';
interface Props {
  sectionTitle: string;
  linkLabel: string;
  buttonHref: string;
  children?: React.ReactNode;
  Svg?: React.FC<IconProps>;
  ariaLabel?: string;
}

export const HomeSection: FC<Props> = ({
  Svg,
  ariaLabel,
  sectionTitle,
  linkLabel,
  buttonHref,
  children
}) => {
  return (
    <Stack border='2px solid' borderColor='primary' h='100%'>
      {Svg && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='primary'>
          <Svg aria-label={ariaLabel} />
        </Stack>
      )}
      <Stack p={4} borderBottom='2px solid' borderColor='primary' sx={{ mt: '0 !important' }}>
        <Box as='h2' textStyle='h2'>
          {sectionTitle}
        </Box>
      </Stack>

      <Stack
        p={4}
        spacing={4}
        borderBottom='2px solid'
        borderColor='primary'
        sx={{ mt: '0 !important' }}
        h='100%'
      >
        {children}
      </Stack>

      <ButtonLinkSecondary href={buttonHref}>{linkLabel}</ButtonLinkSecondary>
    </Stack>
  );
};
