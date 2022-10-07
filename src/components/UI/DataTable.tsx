import {
  Table,
  Thead,
  Tr,
  Th,
  TableContainer,
  Text,
} from '@chakra-ui/react';
import { FC } from 'react';

interface Props {
  columnHeaders: string[]
}

export const DataTable: FC<Props> = ({
  columnHeaders
}) => {
  return (
    <TableContainer>
      <Table variant='unstyled'>
        <Thead>
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
      </Table>
    </TableContainer>
  )
}