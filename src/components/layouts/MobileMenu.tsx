import {
  Box,
  Flex,
  Modal,
  ModalContent,
  ModalOverlay,
  Stack,
  useDisclosure
} from '@chakra-ui/react';
import { CloseIcon } from '@chakra-ui/icons';

import { HeaderButtons, Search } from '../UI';
import { HamburgerIcon } from '../UI/icons';

import { BORDER_WIDTH } from '../../constants';

export const MobileMenu: React.FC = () => {
  const { isOpen, onOpen, onClose } = useDisclosure();

  return (
    <>
      {/* HAMBURGER MENU ICON */}
      <Box
        as='button'
        p={4}
        display={{ base: 'block', md: 'none' }}
        color='primary'
        _hover={{ bg: 'primary', color: 'bg' }}
        onClick={onOpen}
        aria-label='Open mobile menu'
      >
        <HamburgerIcon />
      </Box>

      {/* MODAL */}
      <Modal isOpen={isOpen} onClose={onClose} motionPreset='none'>
        <ModalOverlay />
        <ModalContent>
          {/* MOBILE MENU */}
          <Flex
            position='fixed'
            maxW='min(calc(var(--chakra-sizes-container-sm) - 2rem), 100vw - 2rem)'
            marginInline='auto'
            inset='0'
            top={4}
            mb={4}
            color='bg'
            bg='secondary'
            border={BORDER_WIDTH}
            overflow='hidden'
            direction='column'
            display={{ md: 'none' }}
          >
            <Flex borderBottom={BORDER_WIDTH} justify='flex-end'>
              {/* CLOSE ICON */}
              <Box
                as='button'
                p={4}
                borderInlineStartWidth={BORDER_WIDTH}
                borderColor='bg'
                color='bg'
                _hover={{ bg: 'primary' }}
                onClick={onClose}
                ms='auto'
              >
                <CloseIcon boxSize={5} />
              </Box>
            </Flex>

            {/* HEADER BUTTONS */}
            <HeaderButtons close={onClose} />

            {/* SEARCH */}
            <Stack borderBlockEnd='2px' borderColor='bg'>
              <Search />
            </Stack>
          </Flex>
        </ModalContent>
      </Modal>
    </>
  );
};
