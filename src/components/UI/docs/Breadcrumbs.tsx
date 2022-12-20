import { Breadcrumb, BreadcrumbItem, BreadcrumbLink, Stack } from '@chakra-ui/react';
import NextLink from 'next/link';
import { useRouter } from 'next/router';
import { FC } from 'react';

export const Breadcrumbs: FC = () => {
  const router = useRouter();

  let pathSplit = router.asPath.split('#')[0].split('/');
  pathSplit = pathSplit.splice(1, pathSplit.length);

  return (
    <>
      {router.asPath !== '/docs' && pathSplit.length > 1 ? (
        <Breadcrumb>
          {pathSplit.map((path: string, idx: number) => {
            return (
              <BreadcrumbItem key={path}>
                <NextLink
                  href={`/${pathSplit.slice(0, idx + 1).join('/')}`}
                  passHref
                  legacyBehavior
                >
                  <BreadcrumbLink color={idx + 1 === pathSplit.length ? 'body' : 'primary'}>
                    {path}
                  </BreadcrumbLink>
                </NextLink>
              </BreadcrumbItem>
            );
          })}
        </Breadcrumb>
      ) : (
        <Stack h='24px'></Stack>
      )}
    </>
  );
};
