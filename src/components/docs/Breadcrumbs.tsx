import { Breadcrumb, BreadcrumbItem, BreadcrumbLink } from '@chakra-ui/react';
import NextLink from 'next/link';
import { useRouter } from 'next/router';
import { FC } from 'react';

export const Breadcrumbs: FC = () => {
  const router = useRouter();

  let pathSplit = router.asPath.split('/');
  pathSplit = pathSplit.splice(1, pathSplit.length);

  return (
    <Breadcrumb>
      {pathSplit.map((path: string, idx: number) => {
        return (
          <BreadcrumbItem key={path}>
            <NextLink href={`/${pathSplit.slice(0, idx + 1).join('/')}`} passHref>
              <BreadcrumbLink color={idx + 1 === pathSplit.length ? 'body' : 'primary'}>
                {path}
              </BreadcrumbLink>
            </NextLink>
          </BreadcrumbItem>
        );
      })}
    </Breadcrumb>
  );
};
