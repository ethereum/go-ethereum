import { Stack, Tabs, TabList, Tab, Text, TabPanel, TabPanels } from '@chakra-ui/react';
import { FC } from 'react';

import { DOWNLOAD_TABS, DOWNLOAD_TAB_COLUMN_HEADERS } from '../../../constants';

import { DataTable } from '../../UI';

interface Props {
  data: any;
}

export const DownloadsTable: FC<Props> = ({ data }) => {
  return (
    <Stack sx={{ mt: '0 !important' }} borderBottom='2px solid' borderColor='brand.light.primary'>
      <Tabs variant='unstyled'>
        <TabList color='brand.light.primary' bg='green.50'>
          {DOWNLOAD_TABS.map((tab, idx) => {
            return (
              <Tab
                key={tab}
                w={'20%'}
                p={4}
                _selected={{
                  bg: 'brand.light.primary',
                  color: 'yellow.50'
                }}
                borderBottom='2px solid'
                borderRight={idx === DOWNLOAD_TABS.length - 1 ? 'none' : '2px solid'}
                borderColor='brand.light.primary'
              >
                <Text textStyle='download-tab-label'>{tab}</Text>
              </Tab>
            );
          })}
        </TabList>
        <TabPanels>
          <TabPanel p={0}>
            <DataTable columnHeaders={DOWNLOAD_TAB_COLUMN_HEADERS} data={data} />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable columnHeaders={DOWNLOAD_TAB_COLUMN_HEADERS} data={data} />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable columnHeaders={DOWNLOAD_TAB_COLUMN_HEADERS} data={data} />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable columnHeaders={DOWNLOAD_TAB_COLUMN_HEADERS} data={data} />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable columnHeaders={DOWNLOAD_TAB_COLUMN_HEADERS} data={data} />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Stack>
  );
};
