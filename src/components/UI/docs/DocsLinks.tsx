import { FC } from 'react';
import {
  Accordion,
  AccordionButton,
  AccordionItem,
  AccordionPanel,
  Center,
  Link,
  Stack,
  Text
} from '@chakra-ui/react';
import { AddIcon, MinusIcon } from '@chakra-ui/icons';
import NextLink from 'next/link';

import { LinksList } from './';

import { NavLink } from '../../../types';

interface Props {
  navLinks: NavLink[];
}

export const DocsLinks: FC<Props> = ({ navLinks }) => (
  <Stack border='2px' borderColor='primary'>
    {navLinks.map(({ id, to, items }, idx) => {
      return (
        <Accordion key={id} allowToggle mt='0 !important' defaultIndex={[0]}>
          <AccordionItem border='none'>
            {({ isExpanded }) => (
              <>
                <AccordionButton
                  borderBottom={navLinks.length - 1 === idx ? 'none' : '2px'}
                  p={0}
                  borderColor='primary'
                  justifyContent='space-between'
                  placeContent='flex-end'
                  bg='button-bg'
                >
                  <Stack
                    p={4}
                    borderRight={items ? '2px' : 'none'}
                    borderColor='primary'
                    w='100%'
                    bg='bg'
                  >
                    {to ? (
                      <NextLink href={to} passHref>
                        <Link>
                          <Text textStyle='docs-nav-dropdown'>{id}</Text>
                        </Link>
                      </NextLink>
                    ) : (
                      <Text textStyle='docs-nav-dropdown'>{id}</Text>
                    )}
                  </Stack>

                  {items && (
                    <Stack minW='61px'>
                      <Center>
                        {isExpanded ? (
                          <MinusIcon w='20px' h='20px' color='primary' />
                        ) : (
                          <AddIcon w='20px' h='20px' color='primary' />
                        )}
                      </Center>
                    </Stack>
                  )}
                </AccordionButton>
                {items && (
                  <AccordionPanel borderBottom='2px solid' borderColor='primary' px={0} py={4}>
                    <LinksList links={items} />
                  </AccordionPanel>
                )}
              </>
            )}
          </AccordionItem>
        </Accordion>
      );
    })}
  </Stack>
);
