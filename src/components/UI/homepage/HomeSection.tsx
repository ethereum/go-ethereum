import { Box, Image, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

interface Props {
  imgSrc?: string;
  imgAltText?: string;
  sectionTitle: string;
  linkLabel: string;
  buttonHref: string;
  children?: React.ReactNode;
}

export const HomeSection: FC<Props> = ({
  imgSrc,
  imgAltText,
  sectionTitle,
  linkLabel,
  buttonHref,
  children
}) => {
  return (
    <Stack border='2px solid' borderColor='primary' h='100%'>
      {!!imgSrc && (
        <Stack alignItems='center' p={4} borderBottom='2px solid' borderColor='primary'>
          {/* TODO: use NextImage */}
          <Image src={imgSrc} alt={imgAltText} />
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

      <Stack sx={{ mt: '0 !important' }}>
        <NextLink href={buttonHref} passHref>
          <Link variant='button-link-secondary' isExternal={buttonHref.startsWith('http')}>
            <Text textStyle='home-section-link-label'>{linkLabel}</Text>
          </Link>
        </NextLink>
      </Stack>
    </Stack>
  );
};
