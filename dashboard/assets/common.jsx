// @flow

// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

import {faHome, faLink, faGlobeEurope, faTachometerAlt, faList} from '@fortawesome/free-solid-svg-icons';
import {faCreditCard} from '@fortawesome/free-regular-svg-icons';

type ProvidedMenuProp = {|title: string, icon: string|};
const menuSkeletons: Array<{|id: string, menu: ProvidedMenuProp|}> = [
	{
		id:   'home',
		menu: {
			title: 'Home',
			icon:  faHome,
		},
	}, {
		id:   'chain',
		menu: {
			title: 'Chain',
			icon:  faLink,
		},
	}, {
		id:   'txpool',
		menu: {
			title: 'TxPool',
			icon:  faCreditCard,
		},
	}, {
		id:   'network',
		menu: {
			title: 'Network',
			icon:  faGlobeEurope,
		},
	}, {
		id:   'system',
		menu: {
			title: 'System',
			icon:  faTachometerAlt,
		},
	}, {
		id:   'logs',
		menu: {
			title: 'Logs',
			icon:  faList,
		},
	},
];
export type MenuProp = {|...ProvidedMenuProp, id: string|};
// The sidebar menu and the main content are rendered based on these elements.
// Using the id is circumstantial in some cases, so it is better to insert it also as a value.
// This way the mistyping is prevented.
export const MENU: Map<string, {...MenuProp}> = new Map(menuSkeletons.map(({id, menu}) => ([id, {id, ...menu}])));

export const DURATION = 200;

export const chartStrokeWidth = 0.2;

export const styles = {
	light: {
		color: 'rgba(255, 255, 255, 0.54)',
	},
};

// unit contains the units for the bytePlotter.
export const unit = ['', 'Ki', 'Mi', 'Gi', 'Ti', 'Pi', 'Ei', 'Zi', 'Yi'];

// simplifyBytes returns the simplified version of the given value followed by the unit.
export const simplifyBytes = (x: number) => {
	let i = 0;
	for (; x > 1024 && i < 8; i++) {
		x /= 1024;
	}
	return x.toFixed(2).toString().concat(' ', unit[i], 'B');
};

// hues contains predefined colors for gradient stop colors.
export const hues     = ['#00FF00', '#FFFF00', '#FF7F00', '#FF0000'];
export const hueScale = [0, 2048, 102400, 2097152];
