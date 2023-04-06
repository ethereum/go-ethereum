import { FC, useState } from 'react';
import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Text
} from '@chakra-ui/react';

import { DocsLinks } from './DocsLinks';

import { NavLink } from '../../../types';

interface Props {
  navLinks: NavLink[];
}

export const MobileDocsNav: FC<Props> = ({ navLinks }) => {
  const [isMobileAccordionOpen, setMobileAccordionState] = useState(false);

  const toggleMobileAccordion = () => {
    setMobileAccordionState(prev => !prev);
  };

  return (
    <Accordion allowToggle index={isMobileAccordionOpen ? 0 : -1} onChange={toggleMobileAccordion}>
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
          <Text textStyle='docs-nav-dropdown'>Documentation</Text>
          <AccordionIcon />
        </AccordionButton>

        <AccordionPanel p={0}>
          <DocsLinks navLinks={navLinks} toggleMobileAccordion={toggleMobileAccordion} />
        </AccordionPanel>
      </AccordionItem>
    </Accordion>
  );
};
