import { FC } from 'react';
import { Divider, Link, Stack, Text } from '@chakra-ui/react';
import NextLink from 'next/link';

import { parseHeadingId } from '../../../utils/parseHeadingId';
import { useActiveHash } from '../../../hooks/useActiveHash';

interface Props {
  content: string;
}

export const DocumentNav: FC<Props> = ({ content }) => {
  const parsedHeadings = content
    .split('\n\n')
    .map(item => item.replace(/[\n\r]/g, ''))
    .filter(item => item.startsWith('#'))
    .map(item => parseHeadingId([item]))
    .filter(item => item);

  const activeHash = useActiveHash(parsedHeadings.map(heading => heading!.headingId));

  return (
    <Stack position='sticky' top='4'>
      <Text as='h5' textStyle='document-nav-title'>
        on this page
      </Text>
      <Divider borderColor='primary' my={`4 !important`} />
      {parsedHeadings.map((heading, idx) => {
        return (
          <NextLink key={`${idx} ${heading?.title}`} href={`#${heading?.headingId}`}>
            <Link m={0}>
              <Text
                color={activeHash === heading?.headingId ? 'body' : 'primary'}
                textStyle='document-nav-link'
              >
                {heading?.title}
              </Text>
            </Link>
          </NextLink>
        );
      })}
    </Stack>
  );
};
