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
      // TODO: Work on firefox implementation of scrollbar styles
      // Note: This wont work on safari, we are ok with this.
      css={{
        "&::-webkit-scrollbar": {
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
                  // TODO: Full width for hover
                  // TODO: Add fade in animation on hover
                  // TODO: Get new background color from nuno for hover
                  _hover={{background: 'green.50'}}
                >
                  {
                    columnHeaders.map((columnHeader, idx) => {
                      // TODO: Make the font size smaller (refer to design system)
                      return (
                        <Td
                          key={idx}
                          px={0}
                          pr={8}
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