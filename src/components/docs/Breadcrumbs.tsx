import { Breadcrumb, BreadcrumbItem, BreadcrumbLink } from '@chakra-ui/react';
import NextLink from 'next/link';
import { FC } from 'react';

interface Props {
  router: any;
}

export const Breadcrumbs: FC<Props> = ({ router }) => {
  let pathSplit = router.asPath.split('/');
  pathSplit = pathSplit.splice(1, pathSplit.length);

  return (
    <Breadcrumb mb={10}>
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
