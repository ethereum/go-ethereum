import {
  Stack,
  Tabs,
  TabList,
  Tab,
  Text,
  TabPanel,
  TabPanels,
} from '@chakra-ui/react';

import { DataTable } from '../DataTable'

import { testDownloadData } from '../../../data/test/download-testdata'

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
          <TabPanel p={0}>
            <DataTable
              columnHeaders={[
                'Release',
                'Commit',
                'Kind',
                'Arch',
                'Size',
                'Published'
              ]}
              data={testDownloadData}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={[
                'Release',
                'Commit',
                'Kind',
                'Arch',
                'Size',
                'Published'
              ]}
              data={testDownloadData}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={[
                'Release',
                'Commit',
                'Kind',
                'Arch',
                'Size',
                'Published'
              ]}
              data={testDownloadData}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={[
                'Release',
                'Commit',
                'Kind',
                'Arch',
                'Size',
                'Published'
              ]}
              data={testDownloadData}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={[
                'Release',
                'Commit',
                'Kind',
                'Arch',
                'Size',
                'Published'
              ]}
              data={testDownloadData}
            />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Stack>
  )
}