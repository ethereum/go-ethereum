import { Heading, Image, Link, Stack, Text } from '@chakra-ui/react';
import { FC } from 'react';
import NextLink from 'next/link';

interface Props {
  imgSrc?: string;
  imgAltText?: string;
  sectionTitle: string;
  buttonLabel: string;
  buttonHref: string;
  children?: React.ReactNode;
}

export const HomeSection: FC<Props> = ({
  imgSrc,
  imgAltText,
  sectionTitle,
  buttonLabel,
  buttonHref,
  children
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
        <Heading
          // TODO: move text style to theme
          as='h2'
          fontFamily='"JetBrains Mono", monospace'
          fontWeight={400}
          fontSize='1.5rem'
          lineHeight='auto'
          letterSpacing='4%'
          color='brand.light.body'
        >
          {sectionTitle}
        </Heading>
      </Stack>

      <Stack
        p={4}
        spacing={4}
        borderBottom='2px solid'
        borderColor='brand.light.primary'
        sx={{ mt: '0 !important' }}
      >
        {children}
      </Stack>

      <Stack sx={{ mt: '0 !important' }}>
        <NextLink href={buttonHref} passHref>
          <Link
            color='brand.light.primary'
            bg='green.50'
            _hover={{ textDecoration: 'none', bg: 'brand.light.primary', color: 'yellow.50' }}
            _focus={{
              textDecoration: 'none',
              bg: 'brand.light.primary',
              color: 'yellow.50',
              boxShadow: 'inset 0 0 0 3px #f0f2e2 !important'
            }}
            _active={{ textDecoration: 'none', bg: 'brand.light.secondary', color: 'yellow.50' }}
            isExternal={buttonHref.startsWith('http')}
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              p={4}
            >
              {buttonLabel}
            </Text>
          </Link>
        </NextLink>
      </Stack>
    </Stack>
  );
};
