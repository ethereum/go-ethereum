import {
  Link,
  Table,
  Thead,
  Tr,
  Th,
  TableContainer,
  Text,
  Tbody,
  Td,
  Stack
} from '@chakra-ui/react';
import { FC } from 'react';
import { OpenPGPSignaturesData, ReleaseData } from '../../types';
import { getParsedDate } from '../../utils';

interface Props {
  columnHeaders: string[];
  data: any;
}

export const DataTable: FC<Props> = ({ columnHeaders, data }) => {
  // {} is a backup object for initial render where data is still undefined, to avoid errors
  const dataType = Object.keys(data[0] || {})?.includes('release')
    ? 'Releases'
    : 'OpenPGP Signatures';

  return (
    <TableContainer
      // Note: This wont work on firefox, we are ok with this.
      css={{
        '&::-webkit-scrollbar': {
          borderTop: '2px solid var(--chakra-colors-primary)',
          height: 18
        },
        '&::-webkit-scrollbar-thumb': {
          background: 'var(--chakra-colors-primary)'
        }
      }}
      pt={4}
      pb={4}
    >
      <Table variant='unstyled'>
        {data.length > 0 && (
          <Thead>
            <Tr>
              {columnHeaders.map((columnHeader, idx) => {
                return (
                  <Th key={idx} textTransform='none' minW={'130.5px'} px={4}>
                    <Text
                      fontFamily='"JetBrains Mono", monospace'
                      fontWeight={700}
                      fontSize='md'
                      color='#868b87' // TODO: Use theme color? Or add to theme?
                    >
                      {columnHeader}
                    </Text>
                  </Th>
                );
              })}
            </Tr>
          </Thead>
        )}

        <Tbody>
          {data.length === 0 && (
            <Stack justifyContent='center' alignItems='center' w='100%' minH={80}>
              <Text textStyle='header4'>No builds found</Text>
            </Stack>
          )}

          {dataType === 'Releases' &&
            data.map((r: ReleaseData, idx: number) => {
              const url = r?.release?.url;
              const os = url?.includes('darwin')
                ? 'darwin'
                : url?.includes('linux')
                ? 'linux'
                : url?.includes('windows')
                ? 'windows'
                : url?.includes('android')
                ? 'android'
                : 'ios';

              const isLatestStableLinuxRelease =
                os === 'linux' &&
                data
                  .filter(
                    (e: ReleaseData) => e.arch === '64-bit' && !e.release.url.includes('unstable')
                  )
                  .every((elem: ReleaseData) => {
                    console.log('release:', r);
                    console.log('elem:', elem);

                    return new Date(r.published) >= new Date(elem.published);
                  });

              const x = data.filter((e: ReleaseData, _: any, array: any) => {
                const maxDate = array
                  .map((e: ReleaseData) => new Date(e.published))
                  .filter((f: Date, _: any, array: any) =>
                    array.every((f: Date) => f <= new Date(e.published))
                  );

                return (
                  e.arch === '64-bit' &&
                  !e.release.url.includes('unstable') &&
                  new Date(e.published) === new Date(maxDate)
                );
              });

              console.log(x);

              const latestDarwinRelease = os === 'darwin' && r.arch === '64-bit';
              const latestWindowsRelease = os === 'darwin' && r.kind === 'Installer';
              const latestAndroidRelease = os === 'android' && r.arch === 'all';
              const latestiOSRelease = os === 'ios' && r.arch === 'all';

              // const latest = data.filter(
              //   (rel: ReleaseData) =>
              //     os === 'linux' && rel.arch === '64-bit' && !rel.release.url.includes('unstable')
              // );
              // .every((otherRelease: ReleaseData) => r.published > otherRelease.published);

              // console.log({ latest });

              const isPrimaryRelease =
                isLatestStableLinuxRelease ||
                latestDarwinRelease ||
                latestWindowsRelease ||
                latestAndroidRelease ||
                latestiOSRelease;

              return (
                <Tr
                  key={idx}
                  transition={'all 0.5s'}
                  _hover={{ background: 'button-bg', transition: 'all 0.5s' }}
                  fontWeight={isLatestStableLinuxRelease ? 700 : 400}
                >
                  {Object.entries(r).map((item, idx) => {
                    const objectItems = ['release', 'commit', 'signature'];

                    if (objectItems.includes(item[0])) {
                      const label = item[1].label;
                      const url = item[1].url;

                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Link _hover={{ textDecoration: 'none' }} href={url} isExternal>
                            <Text color='primary'>
                              {item[0] === 'commit' ? `${label}...` : label}
                            </Text>
                          </Link>
                        </Td>
                      );
                    }

                    if (item[0] === 'published') {
                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Text>{getParsedDate(item[1])}</Text>
                        </Td>
                      );
                    }

                    return (
                      <Td key={idx} px={4} textStyle='hero-text-small'>
                        <Text>{item[1]}</Text>
                      </Td>
                    );
                  })}
                </Tr>
              );
            })}

          {dataType === 'OpenPGP Signatures' &&
            data.map((o: OpenPGPSignaturesData, idx: number) => {
              return (
                <Tr
                  key={idx}
                  transition={'all 0.5s'}
                  _hover={{ background: 'button-bg', transition: 'all 0.5s' }}
                >
                  {Object.entries(o).map((item, idx) => {
                    if (item[0] === 'openpgp key') {
                      const label = item[1].label;
                      const url = item[1].url;

                      return (
                        <Td key={idx} px={4} textStyle='hero-text-small'>
                          <Link _hover={{ textDecoration: 'none' }} href={url} isExternal>
                            <Text color='primary'>{label}</Text>
                          </Link>
                        </Td>
                      );
                    }

                    return (
                      <Td key={idx} px={4} textStyle='hero-text-small'>
                        <Text>{item[1]}</Text>
                      </Td>
                    );
                  })}
                </Tr>
              );
            })}
        </Tbody>
      </Table>
    </TableContainer>
  );
};
