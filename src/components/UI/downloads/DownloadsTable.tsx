import {
  Stack,
  Tabs,
  TabList,
  Tab,
  Table,
  Tbody,
  Thead,
  Tr,
  Th,
  Td,
  TableContainer,
  Text,
  TabPanel,
  TabPanels,
} from '@chakra-ui/react';

export const DownloadsTable = () => {
  return (
    <Stack sx={{ mt: '0 !important' }} borderBottom='2px solid #11866f'>
      <Tabs variant='unstyled'>
        <TabList
          color='#11866f'
          bg='#d7f5ef'
          borderBottom='2px solid #11866f'
        >
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: '#11866f',
              color: '#f0f2e2',
            }}
            borderRight='2px solid #11866f'
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              fontSize='sm'
            >
              LINUX
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: '#11866f',
              color: '#f0f2e2',
            }}
            borderRight='2px solid #11866f'
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              fontSize='sm'
            >
              MACOS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: '#11866f',
              color: '#f0f2e2',
            }}
            borderRight='2px solid #11866f'
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              fontSize='sm'
            >
              WINDOWS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: '#11866f',
              color: '#f0f2e2',
            }}
            borderRight='2px solid #11866f'
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              fontSize='sm'
            >
              IOS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: '#11866f',
              color: '#f0f2e2',
            }}
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
              fontSize='sm'
            >
              ANDROID
            </Text>
          </Tab>
        </TabList>
        <TabPanels>
          <TabPanel>
            <TableContainer>
              <Table variant='unstyled'>
                <Thead>
                  <Tr>
                    <Th
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
                        Release
                      </Text>
                    </Th>
                    <Th
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
                        Commit
                      </Text>
                    </Th>
                    <Th
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
                        Kind
                      </Text>
                    </Th>
                    <Th
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
                        Arch
                      </Text>
                    </Th>
                    <Th
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
                        Size
                      </Text>
                    </Th>
                  </Tr>
                </Thead>
              </Table>
            </TableContainer>
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Stack>
  )
}