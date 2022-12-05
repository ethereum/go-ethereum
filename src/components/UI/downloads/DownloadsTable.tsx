import { Stack, Tabs, TabList, Tab, Text, TabPanel, TabPanels } from '@chakra-ui/react';
import { FC } from 'react';

import { DataTable } from '../../UI';

import { DOWNLOADS_TABLE_TABS, DOWNLOADS_TABLE_TAB_COLUMN_HEADERS } from '../../../constants';
import { ReleaseData } from '../../../types';

interface Props {
  linuxData: ReleaseData[];
  macOSData: ReleaseData[];
  windowsData: ReleaseData[];
  iOSData: ReleaseData[];
  androidData: ReleaseData[];
  amountOfReleasesToShow: number;
  setTotalReleases: (idx: number) => void;
}

export const DownloadsTable: FC<Props> = ({
  linuxData,
  macOSData,
  windowsData,
  iOSData,
  androidData,
  amountOfReleasesToShow,
  setTotalReleases
}) => {
  const totalReleases = [
    linuxData.length,
    macOSData.length,
    windowsData.length,
    iOSData.length,
    androidData.length
  ];

  const LAST_2_LINUX_RELEASES = amountOfReleasesToShow + 12;

  return (
    <Stack sx={{ mt: '0 !important' }} borderBottom='2px solid' borderColor='primary'>
      <Tabs variant='unstyled' onChange={idx => setTotalReleases(totalReleases[idx])}>
        <TabList color='primary' bg='button-bg'>
          {DOWNLOADS_TABLE_TABS.map((tab, idx) => {
            return (
              <Tab
                key={tab}
                w={'20%'}
                p={4}
                _selected={{
                  bg: 'primary',
                  color: 'bg'
                }}
                borderBottom='2px solid'
                borderRight={idx === DOWNLOADS_TABLE_TABS.length - 1 ? 'none' : '2px solid'}
                borderColor='primary'
              >
                <Text textStyle='download-tab-label'>{tab}</Text>
              </Tab>
            );
          })}
        </TabList>

        <TabPanels>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={DOWNLOADS_TABLE_TAB_COLUMN_HEADERS}
              data={linuxData.slice(0, LAST_2_LINUX_RELEASES)}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={DOWNLOADS_TABLE_TAB_COLUMN_HEADERS}
              data={macOSData.slice(0, amountOfReleasesToShow)}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={DOWNLOADS_TABLE_TAB_COLUMN_HEADERS}
              data={windowsData.slice(0, amountOfReleasesToShow)}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={DOWNLOADS_TABLE_TAB_COLUMN_HEADERS}
              data={iOSData.slice(0, amountOfReleasesToShow)}
            />
          </TabPanel>
          <TabPanel p={0}>
            <DataTable
              columnHeaders={DOWNLOADS_TABLE_TAB_COLUMN_HEADERS}
              data={androidData.slice(0, amountOfReleasesToShow)}
            />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Stack>
  );
};
