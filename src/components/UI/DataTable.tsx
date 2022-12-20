import {
  Link,
  Table,
  Thead,
  Tr,
  Th,
  TableContainer,
  Text,
  Tbody,
  Td,
  Stack
} from '@chakra-ui/react';
import { FC } from 'react';

import {
  getOS,
  getParsedDate,
  isDarwinPrimaryRelease,
  isLinuxPrimaryRelease,
  isMobilePrimaryRelease,
  isWindowsPrimaryRelease
} from '../../utils';
import { OpenPGPSignaturesData, ReleaseData } from '../../types';

interface Props {
  columnHeaders: string[];
  data: any;
}

export const DataTable: FC<Props> = ({ columnHeaders, data }) => {
  // {} is a backup object for initial render where data is still undefined, to avoid errors
  const dataType = Object.keys(data[0] || {})?.includes('release')
    ? 'Releases'
    : 'OpenPGP Signatures';

  return (
    <TableContainer
      // Note: This wont work on firefox, we are ok with this.
      css={{
        '&::-webkit-scrollbar': {
          borderTop: '2px solid var(--chakra-colors-primary)',
          height: 18
        },
        '&::-webkit-scrollbar-thumb': {
          background: 'var(--chakra-colors-primary)'
        }
      }}
      pt={4}
      pb={4}
    >
      <Table variant='unstyled'>
        {data.length > 0 && (
          <Thead>
            <Tr>
              {columnHeaders.map((columnHeader, idx) => {
                return (
                  <Th key={idx} textTransform='none' minW={'130.5px'} px={4}>
                    <Text
                      fontFamily='"JetBrains Mono", monospace'
                      fontWeight={700}
                      fontSize='md'
                      color='#868b87' // TODO: Use theme color? Or add to theme?
                    >
                      {columnHeader}
                    </Text>
                  </Th>
                );
              })}
            </Tr>
          </Thead>
        )}

        <Tbody>
          {data.length === 0 && (
            <Stack justifyContent='center' alignItems='center' w='100%' minH={80}>
              <Text textStyle='header4'>No builds found</Text>
            </Stack>
          )}

          {dataType === 'Releases' &&
            data.map((r: ReleaseData, idx: number) => {
              const url = r?.release?.url;
              const os = getOS(url);

              const _isLinuxPrimaryRelease = isLinuxPrimaryRelease(r, os, data);
              const _isDarwinPrimaryRelease = isDarwinPrimaryRelease(r, os, data);
              const _isWindowsPrimaryRelease = isWindowsPrimaryRelease(r, os, data);
              const _isMobilePrimaryRelease = isMobilePrimaryRelease(r, os, data);

              const isPrimaryRelease =
                _isLinuxPrimaryRelease ||
                _isDarwinPrimaryRelease ||
                _isWindowsPrimaryRelease ||
                _isMobilePrimaryRelease;

              return (
                <Tr
                  key={idx}
                  transition={'all 0.5s'}
                  _hover={{ background: 'button-bg', transition: 'all 0.5s' }}
                  fontWeight={isPrimaryRelease ? 700 : 400}
                >
                  {Object.entries(r).map((item, idx) => {
                    const objectItems = ['release', 'commit', 'signature'];

                    if (objectItems.includes(item[0])) {
                      const label = item[1].label;
                      const url = item[1].url;

                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Link _hover={{ textDecoration: 'none' }} href={url} isExternal>
                            <Text color='primary'>
                              {item[0] === 'commit' ? `${label}...` : label}
                            </Text>
                          </Link>
                        </Td>
                      );
                    }

                    if (item[0] === 'published') {
                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Text>{getParsedDate(item[1])}</Text>
                        </Td>
                      );
                    }

                    return (
                      <Td key={idx} px={4} textStyle='hero-text-small'>
                        <Text>{item[1]}</Text>
                      </Td>
                    );
                  })}
                </Tr>
              );
            })}

          {dataType === 'OpenPGP Signatures' &&
            data.map((o: OpenPGPSignaturesData, idx: number) => {
              return (
                <Tr
                  key={idx}
                  transition={'all 0.5s'}
                  _hover={{ background: 'button-bg', transition: 'all 0.5s' }}
                >
                  {Object.entries(o).map((item, idx) => {
                    if (item[0] === 'openpgp key') {
                      const label = item[1].label;
                      const url = item[1].url;

                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Link _hover={{ textDecoration: 'none' }} href={url} isExternal>
                            <Text color='primary'>{label}</Text>
                          </Link>
                        </Td>
                      );
                    }

                    return (
                      <Td key={idx} px={4} textStyle='hero-text-small'>
                        <Text>{item[1]}</Text>
                      </Td>
                    );
                  })}
                </Tr>
              );
            })}
        </Tbody>
      </Table>
    </TableContainer>
  );
};
