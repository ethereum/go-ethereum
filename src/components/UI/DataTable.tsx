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
      // Note: This wont work on firefox, we are ok with this.
      css={{
        "&::-webkit-scrollbar": {
          borderTop: '2px solid #11866f',
          height: 18
        },
        "&::-webkit-scrollbar-thumb": {
          background: "#11866f",
        },
      }}
      pt={4}
      pb={4}
    >
      <Table variant='unstyled'>
        <Thead>
          <Tr>
            {
              columnHeaders.map((columnHeader, idx) => {
                return (
                  <Th
                    key={idx}
                    textTransform='none'
                    minW={'130.5px'}
                    px={4}
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
                  // TODO: Get new background color from nuno for hover
                  transition={'all 0.5s'}
                  _hover={{background: 'green.50', transition: 'all 0.5s'}}
                >
                  {
                    columnHeaders.map((columnHeader, idx) => {
                      // TODO: Make the font size smaller (refer to design system)
                      return (
                        <Td
                          key={idx}
                          px={4}
                          fontSize='13px'
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