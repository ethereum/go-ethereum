import { FC, useState } from 'react';
import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Stack,
  Text
} from '@chakra-ui/react';
import { DocsLinks } from './DocsLinks';

import { NavLink } from '../../../types';

interface Props {
  navLinks: NavLink[];
}

export const DocsNav: FC<Props> = ({ navLinks }) => {
  const OPEN = 0;
  const CLOSED = -1;
  const [mobileAccordionState, setMobileAccordionState] = useState(CLOSED)

  const updateMobileAccordionState = () => {
    setMobileAccordionState(mobileAccordionState === OPEN ? CLOSED : OPEN)
  }

  return (
    <Stack w={{ base: '100%', lg: 72 }}>
      <Stack display={{ base: 'none', lg: 'block' }}>
        <DocsLinks navLinks={navLinks} updateMobileAccordionState={updateMobileAccordionState} />
      </Stack>

      <Stack display={{ base: 'block', lg: 'none' }}>
        <Accordion allowToggle index={mobileAccordionState} onChange={updateMobileAccordionState}>
          <AccordionItem border='none'>
            <AccordionButton
              display='flex'
              py={4}
              px={8}
              border='2px'
              borderColor='primary'
              placeContent='space-between'
              bg='button-bg'
              _hover={{
                bg: 'primary',
                color: 'bg'
              }}
              _expanded={{
                bg: 'primary',
                color: 'bg'
              }}
            >
              <Text as='h4' textStyle='docs-nav-dropdown'>
                Documentation
              </Text>
              <AccordionIcon />
            </AccordionButton>
            <AccordionPanel p={0}>
              <DocsLinks navLinks={navLinks} updateMobileAccordionState={updateMobileAccordionState} />
            </AccordionPanel>
          </AccordionItem>
        </Accordion>
      </Stack>
    </Stack>
  );
};
