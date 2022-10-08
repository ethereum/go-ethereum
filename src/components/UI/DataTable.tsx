import {
  Table,
  Thead,
  Tr,
  Th,
  TableContainer,
  Text,
  Tbody,
  Td,
} from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  columnHeaders: string[]
  data: any
}

export const DataTable: FC<Props> = ({
  columnHeaders,
  data,
}) => {
  return (
    <TableContainer
      css={{
        "&::-webkit-scrollbar": {
          background: "#d7f5ef",
          borderTop: '2px solid #11866f',
          height: 18
        },
        "&::-webkit-scrollbar-thumb": {
          background: "#11866f",
        },
      }}
      p={4}
    >
      <Table
        variant='unstyled'
      >
        <Thead
        >
          <Tr>
            {
              columnHeaders.map((columnHeader, idx) => {
                return (
                  <Th
                    key={idx}
                    textTransform='none'
                    p={0}
                    minW={'130.5px'}
                  >
                    <Text
                      fontFamily='"JetBrains Mono", monospace'
                      fontWeight={700}
                      fontSize='md'
                      color='#868b87'
                    >
                      {columnHeader}
                    </Text>
                  </Th>
                )
              })
            }
          </Tr>
        </Thead>
        <Tbody>
          {
            data.map((item: any, idx: number) => {
              return (
                <Tr
                  key={idx}
                >
                  {
                    columnHeaders.map((columnHeader, idx) => {
                      return (
                        <Td
                          key={idx}
                          px={0}
                          pr={2}
                        >
                          {item[columnHeader.toLowerCase()]}
                        </Td>
                      )
                    })
                  }
                </Tr>
              )
            })
          }
        </Tbody>
      </Table>
    </TableContainer>
  )
}