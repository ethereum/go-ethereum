import { ReactNode, createContext, useState } from 'react';

import { NavLink } from '../types';

export interface NavLinksContextInterface {
  _navLinks: NavLink[];
  setNavLinks: (navLinks: NavLink[]) => void;
}

const defaultState = {
  _navLinks: [],
  setNavLinks: (mobileNavLinks: NavLink[]) => {}
};

// initialize Context with default state
export const NavLinksContext = createContext<NavLinksContextInterface>(defaultState);

interface Props {
  children: ReactNode;
}

export const NavLinksContextProvider = ({ children }: Props) => {
  const [_navLinks, setNavLinks] = useState<NavLink[]>([]);

  return (
    <NavLinksContext.Provider value={{ _navLinks, setNavLinks }}>
      {children}
    </NavLinksContext.Provider>
  );
};
