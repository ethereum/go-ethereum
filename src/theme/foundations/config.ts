import { type ThemeConfig } from '@chakra-ui/react';
/**
 * https://chakra-ui.com/docs/styled-system/color-mode
 * initialColorMode: 'system' —— Will default to users system color mode
 * useSystemColorMode=true —— Color mode will change if user changes their system color mode
 * Can be overridden with toggle on site and will persist after refresh
 * Choice is stored/managed with local storage
 */
export const config: ThemeConfig = {
  initialColorMode: 'system',
  useSystemColorMode: true
};
