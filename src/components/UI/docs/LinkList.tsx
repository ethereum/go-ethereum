import { FC } from 'react';
import { Link, Stack, Text } from '@chakra-ui/react';
import NextLink from 'next/link';

import { NavLink } from '../../../types';

interface LinksListProps {
  links: NavLink[];
}

export const LinksList: FC<LinksListProps> = ({ links }) => (
  <Stack px={4}>
    {links.map(({ id, to, items }) => {
      return to ? (
        <Stack key={id}>
          <NextLink href={to} passHref key={id}>
            <Link>
              <Text textStyle='docs-nav-links' color={items ? 'primary' : 'body'}>
                {id}
              </Text>
            </Link>
          </NextLink>
          {items && <LinksList links={items} />}
        </Stack>
      ) : (
        <Stack key={id}>
          <Text textStyle='docs-nav-links' color={items ? 'primary' : 'body'}>
            {id}
          </Text>
          {items && <LinksList links={items} />}
        </Stack>
      );
    })}
  </Stack>
);
