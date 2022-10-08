import {
  Stack,
  Tabs,
  TabList,
  Tab,
  Text,
  TabPanel,
  TabPanels,
} from '@chakra-ui/react';
import { FC } from 'react';

import { DataTable } from '../DataTable'

interface Props {
  data: any
}

export const DownloadsTable: FC<Props> = ({
  data
}) => {
  return (
    <Stack sx={{ mt: '0 !important' }} borderBottom='2px solid #11866f'>
      <Tabs variant='unstyled'>
        <TabList
          color='brand.light.primary'
          bg='green.50'
        >
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: 'brand.light.primary',
              color: 'yellow.50',
            }}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
          >
            <Text textStyle='download-tab-label'>
              LINUX
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: 'brand.light.primary',
              color: 'yellow.50',
            }}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
          >
            <Text textStyle='download-tab-label'>
              MACOS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: 'brand.light.primary',
              color: 'yellow.50',
            }}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
          >
            <Text textStyle='download-tab-label'>
              WINDOWS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: 'brand.light.primary',
              color: 'yellow.50',
            }}
            borderRight='2px solid'
            borderBottom='2px solid'
            borderColor='brand.light.primary'
          >
            <Text textStyle='download-tab-label'>
              IOS
            </Text>
          </Tab>
          <Tab
            w={'20%'}
            p={4}
            _selected={{
              bg: 'brand.light.primary',
              color: 'yellow.50',
            }}
            borderBottom='2px solid'
            borderColor='brand.light.primary'
          >
            <Text textStyle='download-tab-label'>
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
              data={data}
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
              data={data}
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
              data={data}
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
              data={data}
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
              data={data}
            />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Stack>
  )
}