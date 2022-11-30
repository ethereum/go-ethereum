import { Link, Table, Thead, Tr, Th, TableContainer, Text, Tbody, Td } from '@chakra-ui/react';
import { FC } from 'react';
import { ReleaseData } from '../../types';
import { getParsedDate } from '../../utils';

interface Props {
  columnHeaders: string[];
  // TODO: update data type
  data: any;
}

export const DataTable: FC<Props> = ({ columnHeaders, data }) => {
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
        <Thead>
          <Tr>
            {columnHeaders.map((columnHeader, idx) => {
              return (
                <Th key={idx} textTransform='none' minW={'130.5px'} px={4}>
                  <Text
                    fontFamily='"JetBrains Mono", monospace'
                    fontWeight={700}
                    fontSize='md'
                    color='#868b87' //? Use theme color? Or add to theme?
                  >
                    {columnHeader}
                  </Text>
                </Th>
              );
            })}
          </Tr>
        </Thead>

        <Tbody>
          {data.map((r: ReleaseData, idx: number) => {
            return (
              <Tr
                key={idx}
                transition={'all 0.5s'}
                _hover={{ background: 'button-bg', transition: 'all 0.5s' }}
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
        </Tbody>
      </Table>
    </TableContainer>
  );
};
