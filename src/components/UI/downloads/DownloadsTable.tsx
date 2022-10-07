import {
  Stack,
  Tabs,
  TabList,
  Tab,
  Text,
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
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
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
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
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
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
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
          >
            <Text
              fontFamily='"JetBrains Mono", monospace'
              // TODO: move to theme colors
              fontWeight={700}
              textTransform='uppercase'
              textAlign='center'
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
            >
              ANDROID
            </Text>
          </Tab>
        </TabList>
      </Tabs>
      <Text>Test</Text>
    </Stack>
  )
}