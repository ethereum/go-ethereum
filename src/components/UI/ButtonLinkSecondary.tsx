import { Link, Stack, Text } from '@chakra-ui/react';
import NextLink, { LinkProps } from 'next/link';

import { Link as LinkTheme } from '../../theme/components';

interface Props extends LinkProps {
  children: React.ReactNode;
}

export const ButtonLinkSecondary: React.FC<Props> = ({ href, children, ...restProps }) => {
  const isExternal: boolean = href.toString().startsWith('http');

  const variant = LinkTheme.variants['button-link-secondary'];
  return (
    <Stack sx={{ mt: '0 !important' }} {...variant}>
      {isExternal ? (
        <Link variant='button-link-secondary' href={href.toString()} isExternal>
          <Text textStyle='home-section-link-label'>{children}</Text>
        </Link>
      ) : (
        <NextLink href={href.toString()} passHref legacyBehavior {...restProps}>
          <Link variant='button-link-secondary'>
            <Text textStyle='home-section-link-label'>{children}</Text>
          </Link>
        </NextLink>
      )}
    </Stack>
  );
};
