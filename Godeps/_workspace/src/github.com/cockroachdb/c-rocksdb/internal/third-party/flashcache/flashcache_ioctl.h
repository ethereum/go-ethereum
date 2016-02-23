/****************************************************************************
 *  flashcache_ioctl.h
 *  FlashCache: Device mapper target for block-level disk caching
 *
 *  Copyright 2010 Facebook, Inc.
 *  Author: Mohan Srinivasan (mohan@facebook.com)
 *
 *  Based on DM-Cache:
 *   Copyright (C) International Business Machines Corp., 2006
 *   Author: Ming Zhao (mingzhao@ufl.edu)
 *
 *  This program is free software; you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation; under version 2 of the License.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 ****************************************************************************/

#ifdef OS_LINUX
#ifndef FLASHCACHE_IOCTL_H
#define FLASHCACHE_IOCTL_H

#include <linux/types.h>

#define FLASHCACHE_IOCTL 0xfe

enum {
	FLASHCACHEADDNCPID_CMD=200,
	FLASHCACHEDELNCPID_CMD,
	FLASHCACHEDELNCALL_CMD,
	FLASHCACHEADDWHITELIST_CMD,
	FLASHCACHEDELWHITELIST_CMD,
	FLASHCACHEDELWHITELISTALL_CMD,
};

#define FLASHCACHEADDNCPID	_IOW(FLASHCACHE_IOCTL, FLASHCACHEADDNCPID_CMD, pid_t)
#define FLASHCACHEDELNCPID	_IOW(FLASHCACHE_IOCTL, FLASHCACHEDELNCPID_CMD, pid_t)
#define FLASHCACHEDELNCALL	_IOW(FLASHCACHE_IOCTL, FLASHCACHEDELNCALL_CMD, pid_t)

#define FLASHCACHEADDBLACKLIST		FLASHCACHEADDNCPID
#define FLASHCACHEDELBLACKLIST		FLASHCACHEDELNCPID
#define FLASHCACHEDELALLBLACKLIST	FLASHCACHEDELNCALL

#define FLASHCACHEADDWHITELIST		_IOW(FLASHCACHE_IOCTL, FLASHCACHEADDWHITELIST_CMD, pid_t)
#define FLASHCACHEDELWHITELIST		_IOW(FLASHCACHE_IOCTL, FLASHCACHEDELWHITELIST_CMD, pid_t)
#define FLASHCACHEDELALLWHITELIST	_IOW(FLASHCACHE_IOCTL, FLASHCACHEDELWHITELISTALL_CMD, pid_t)

#endif /* FLASHCACHE_IOCTL_H */
#endif /* OS_LINUX */
