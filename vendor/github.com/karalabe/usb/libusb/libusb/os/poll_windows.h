/*
 * Windows compat: POSIX compatibility wrapper
 * Copyright © 2012-2013 RealVNC Ltd.
 * Copyright © 2009-2010 Pete Batard <pete@akeo.ie>
 * Copyright © 2016-2018 Chris Dickens <christopher.a.dickens@gmail.com>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of poll implementation from libusb-win32, by Stephan Meyer et al.
 *
 * This library is free software; you can redistribute it and/or
 * modify it under the terms of the GNU Lesser General Public
 * License as published by the Free Software Foundation; either
 * version 2.1 of the License, or (at your option) any later version.
 *
 * This library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public
 * License along with this library; if not, write to the Free Software
 * Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA
 *
 */
#pragma once

#if defined(_MSC_VER)
// disable /W4 MSVC warnings that are benign
#pragma warning(disable:4127) // conditional expression is constant
#endif

// Handle synchronous completion through the overlapped structure
#if !defined(STATUS_REPARSE)	// reuse the REPARSE status code
#define STATUS_REPARSE ((LONG)0x00000104L)
#endif
#define STATUS_COMPLETED_SYNCHRONOUSLY	STATUS_REPARSE
#if defined(_WIN32_WCE)
// WinCE doesn't have a HasOverlappedIoCompleted() macro, so attempt to emulate it
#define HasOverlappedIoCompleted(lpOverlapped) (((DWORD)(lpOverlapped)->Internal) != STATUS_PENDING)
#endif
#define HasOverlappedIoCompletedSync(lpOverlapped)	(((DWORD)(lpOverlapped)->Internal) == STATUS_COMPLETED_SYNCHRONOUSLY)

#define DUMMY_HANDLE ((HANDLE)(LONG_PTR)-2)

#define MAX_FDS     256

#define POLLIN      0x0001    /* There is data to read */
#define POLLPRI     0x0002    /* There is urgent data to read */
#define POLLOUT     0x0004    /* Writing now will not block */
#define POLLERR     0x0008    /* Error condition */
#define POLLHUP     0x0010    /* Hung up */
#define POLLNVAL    0x0020    /* Invalid request: fd not open */

struct pollfd {
	int fd;		/* file descriptor */
	short events;	/* requested events */
	short revents;	/* returned events */
};

struct winfd {
	int fd;				// what's exposed to libusb core
	OVERLAPPED *overlapped;		// what will report our I/O status
};

extern const struct winfd INVALID_WINFD;

struct winfd usbi_create_fd(void);

int usbi_pipe(int pipefd[2]);
int usbi_poll(struct pollfd *fds, unsigned int nfds, int timeout);
ssize_t usbi_write(int fd, const void *buf, size_t count);
ssize_t usbi_read(int fd, void *buf, size_t count);
int usbi_close(int fd);

/*
 * Timeval operations
 */
#if defined(DDKBUILD)
#include <winsock.h>	// defines timeval functions on DDK
#endif

#if !defined(TIMESPEC_TO_TIMEVAL)
#define TIMESPEC_TO_TIMEVAL(tv, ts) {                   \
	(tv)->tv_sec = (long)(ts)->tv_sec;                  \
	(tv)->tv_usec = (long)(ts)->tv_nsec / 1000;         \
}
#endif
#if !defined(timersub)
#define timersub(a, b, result)                          \
do {                                                    \
	(result)->tv_sec = (a)->tv_sec - (b)->tv_sec;       \
	(result)->tv_usec = (a)->tv_usec - (b)->tv_usec;    \
	if ((result)->tv_usec < 0) {                        \
		--(result)->tv_sec;                             \
		(result)->tv_usec += 1000000;                   \
	}                                                   \
} while (0)
#endif
