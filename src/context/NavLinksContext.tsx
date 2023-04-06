import { ReactNode, createContext, useState } from 'react';

import { NavLink } from '../types';

export interface NavLinksContextInterface {
  mobileNavLinks: NavLink[];
  setMobileNavLinks: (navLinks: NavLink[]) => void;
}

const defaultState = {
  mobileNavLinks: [],
  setMobileNavLinks: (mobileNavLinks: NavLink[]) => {}
};

// initialize Context with default state
export const NavLinksContext = createContext<NavLinksContextInterface>(defaultState);

interface Props {
  children: ReactNode;
}

export const NavLinksContextProvider = ({ children }: Props) => {
  const [mobileNavLinks, setMobileNavLinks] = useState<NavLink[]>([]);

  return (
    <NavLinksContext.Provider value={{ mobileNavLinks, setMobileNavLinks }}>
      {children}
    </NavLinksContext.Provider>
  );
};
