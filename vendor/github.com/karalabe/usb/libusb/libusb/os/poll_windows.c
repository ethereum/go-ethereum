/*
 * poll_windows: poll compatibility wrapper for Windows
 * Copyright © 2017 Chris Dickens <christopher.a.dickens@gmail.com>
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

/*
 * poll() and pipe() Windows compatibility layer for libusb 1.0
 *
 * The way this layer works is by using OVERLAPPED with async I/O transfers, as
 * OVERLAPPED have an associated event which is flagged for I/O completion.
 *
 * For USB pollable async I/O, you would typically:
 * - obtain a Windows HANDLE to a file or device that has been opened in
 *   OVERLAPPED mode
 * - call usbi_create_fd with this handle to obtain a custom fd.
 * - leave the core functions call the poll routine and flag POLLIN/POLLOUT
 *
 * The pipe pollable synchronous I/O works using the overlapped event associated
 * with a fake pipe. The read/write functions are only meant to be used in that
 * context.
 */
#include <config.h>

#include <assert.h>
#include <errno.h>
#include <stdlib.h>

#include "libusbi.h"
#include "windows_common.h"

// public fd data
const struct winfd INVALID_WINFD = { -1, NULL };

// private data
struct file_descriptor {
	enum fd_type { FD_TYPE_PIPE, FD_TYPE_TRANSFER } type;
	OVERLAPPED overlapped;
};

static usbi_mutex_static_t fd_table_lock = USBI_MUTEX_INITIALIZER;
static struct file_descriptor *fd_table[MAX_FDS];

static struct file_descriptor *create_fd(enum fd_type type)
{
	struct file_descriptor *fd = calloc(1, sizeof(*fd));
	if (fd == NULL)
		return NULL;
	fd->overlapped.hEvent = CreateEvent(NULL, TRUE, FALSE, NULL);
	if (fd->overlapped.hEvent == NULL) {
		free(fd);
		return NULL;
	}
	fd->type = type;
	return fd;
}

static void free_fd(struct file_descriptor *fd)
{
	CloseHandle(fd->overlapped.hEvent);
	free(fd);
}

/*
 * Create both an fd and an OVERLAPPED, so that it can be used with our
 * polling function
 * The handle MUST support overlapped transfers (usually requires CreateFile
 * with FILE_FLAG_OVERLAPPED)
 * Return a pollable file descriptor struct, or INVALID_WINFD on error
 *
 * Note that the fd returned by this function is a per-transfer fd, rather
 * than a per-session fd and cannot be used for anything else but our
 * custom functions.
 * if you plan to do R/W on the same handle, you MUST create 2 fds: one for
 * read and one for write. Using a single R/W fd is unsupported and will
 * produce unexpected results
 */
struct winfd usbi_create_fd(void)
{
	struct file_descriptor *fd;
	struct winfd wfd;

	fd = create_fd(FD_TYPE_TRANSFER);
	if (fd == NULL)
		return INVALID_WINFD;

	usbi_mutex_static_lock(&fd_table_lock);
	for (wfd.fd = 0; wfd.fd < MAX_FDS; wfd.fd++) {
		if (fd_table[wfd.fd] != NULL)
			continue;
		fd_table[wfd.fd] = fd;
		break;
	}
	usbi_mutex_static_unlock(&fd_table_lock);

	if (wfd.fd == MAX_FDS) {
		free_fd(fd);
		return INVALID_WINFD;
	}

	wfd.overlapped = &fd->overlapped;

	return wfd;
}

static int check_pollfds(struct pollfd *fds, unsigned int nfds,
	HANDLE *wait_handles, DWORD *nb_wait_handles)
{
	struct file_descriptor *fd;
	unsigned int n;
	int nready = 0;

	usbi_mutex_static_lock(&fd_table_lock);

	for (n = 0; n < nfds; ++n) {
		fds[n].revents = 0;

		// Keep it simple - only allow either POLLIN *or* POLLOUT
		assert((fds[n].events == POLLIN) || (fds[n].events == POLLOUT));
		if ((fds[n].events != POLLIN) && (fds[n].events != POLLOUT)) {
			fds[n].revents = POLLNVAL;
			nready++;
			continue;
		}

		if ((fds[n].fd >= 0) && (fds[n].fd < MAX_FDS))
			fd = fd_table[fds[n].fd];
		else
			fd = NULL;

		assert(fd != NULL);
		if (fd == NULL) {
			fds[n].revents = POLLNVAL;
			nready++;
			continue;
		}

		if (HasOverlappedIoCompleted(&fd->overlapped)
				&& (WaitForSingleObject(fd->overlapped.hEvent, 0) == WAIT_OBJECT_0)) {
			fds[n].revents = fds[n].events;
			nready++;
		} else if (wait_handles != NULL) {
			if (*nb_wait_handles == MAXIMUM_WAIT_OBJECTS) {
				usbi_warn(NULL, "too many HANDLEs to wait on");
				continue;
			}
			wait_handles[*nb_wait_handles] = fd->overlapped.hEvent;
			(*nb_wait_handles)++;
		}
	}

	usbi_mutex_static_unlock(&fd_table_lock);

	return nready;
}
/*
 * POSIX poll equivalent, using Windows OVERLAPPED
 * Currently, this function only accepts one of POLLIN or POLLOUT per fd
 * (but you can create multiple fds from the same handle for read and write)
 */
int usbi_poll(struct pollfd *fds, unsigned int nfds, int timeout)
{
	HANDLE wait_handles[MAXIMUM_WAIT_OBJECTS];
	DWORD nb_wait_handles = 0;
	DWORD ret;
	int nready;

	nready = check_pollfds(fds, nfds, wait_handles, &nb_wait_handles);

	// If nothing was triggered, wait on all fds that require it
	if ((nready == 0) && (nb_wait_handles != 0) && (timeout != 0)) {
		ret = WaitForMultipleObjects(nb_wait_handles, wait_handles,
			FALSE, (timeout < 0) ? INFINITE : (DWORD)timeout);
		if (ret < (WAIT_OBJECT_0 + nb_wait_handles)) {
			nready = check_pollfds(fds, nfds, NULL, NULL);
		} else if (ret != WAIT_TIMEOUT) {
			if (ret == WAIT_FAILED)
				usbi_err(NULL, "WaitForMultipleObjects failed: %u", (unsigned int)GetLastError());
			nready = -1;
		}
	}

	return nready;
}

/*
 * close a fake file descriptor
 */
int usbi_close(int _fd)
{
	struct file_descriptor *fd;

	if (_fd < 0 || _fd >= MAX_FDS)
		goto err_badfd;

	usbi_mutex_static_lock(&fd_table_lock);
	fd = fd_table[_fd];
	fd_table[_fd] = NULL;
	usbi_mutex_static_unlock(&fd_table_lock);

	if (fd == NULL)
		goto err_badfd;

	if (fd->type == FD_TYPE_PIPE) {
		// InternalHigh is our reference count
		fd->overlapped.InternalHigh--;
		if (fd->overlapped.InternalHigh == 0)
			free_fd(fd);
	} else {
		free_fd(fd);
	}

	return 0;

err_badfd:
	errno = EBADF;
	return -1;
}

/*
* Create a fake pipe.
* As libusb only uses pipes for signaling, all we need from a pipe is an
* event. To that extent, we create a single wfd and overlapped as a means
* to access that event.
*/
int usbi_pipe(int filedes[2])
{
	struct file_descriptor *fd;
	int r_fd = -1, w_fd = -1;
	int i;

	fd = create_fd(FD_TYPE_PIPE);
	if (fd == NULL) {
		errno = ENOMEM;
		return -1;
	}

	// Use InternalHigh as a reference count
	fd->overlapped.Internal = STATUS_PENDING;
	fd->overlapped.InternalHigh = 2;

	usbi_mutex_static_lock(&fd_table_lock);
	do {
		for (i = 0; i < MAX_FDS; i++) {
			if (fd_table[i] != NULL)
				continue;
			if (r_fd == -1) {
				r_fd = i;
			} else if (w_fd == -1) {
				w_fd = i;
				break;
			}
		}

		if (i == MAX_FDS)
			break;

		fd_table[r_fd] = fd;
		fd_table[w_fd] = fd;

	} while (0);
	usbi_mutex_static_unlock(&fd_table_lock);

	if (i == MAX_FDS) {
		free_fd(fd);
		errno = EMFILE;
		return -1;
	}

	filedes[0] = r_fd;
	filedes[1] = w_fd;

	return 0;
}

/*
 * synchronous write for fake "pipe" signaling
 */
ssize_t usbi_write(int fd, const void *buf, size_t count)
{
	int error = EBADF;

	UNUSED(buf);

	if (fd < 0 || fd >= MAX_FDS)
		goto err_out;

	if (count != sizeof(unsigned char)) {
		usbi_err(NULL, "this function should only used for signaling");
		error = EINVAL;
		goto err_out;
	}

	usbi_mutex_static_lock(&fd_table_lock);
	if ((fd_table[fd] != NULL) && (fd_table[fd]->type == FD_TYPE_PIPE)) {
		assert(fd_table[fd]->overlapped.Internal == STATUS_PENDING);
		assert(fd_table[fd]->overlapped.InternalHigh == 2);
		fd_table[fd]->overlapped.Internal = STATUS_WAIT_0;
		SetEvent(fd_table[fd]->overlapped.hEvent);
		error = 0;
	}
	usbi_mutex_static_unlock(&fd_table_lock);

	if (error)
		goto err_out;

	return sizeof(unsigned char);

err_out:
	errno = error;
	return -1;
}

/*
 * synchronous read for fake "pipe" signaling
 */
ssize_t usbi_read(int fd, void *buf, size_t count)
{
	int error = EBADF;

	UNUSED(buf);

	if (fd < 0 || fd >= MAX_FDS)
		goto err_out;

	if (count != sizeof(unsigned char)) {
		usbi_err(NULL, "this function should only used for signaling");
		error = EINVAL;
		goto err_out;
	}

	usbi_mutex_static_lock(&fd_table_lock);
	if ((fd_table[fd] != NULL) && (fd_table[fd]->type == FD_TYPE_PIPE)) {
		assert(fd_table[fd]->overlapped.Internal == STATUS_WAIT_0);
		assert(fd_table[fd]->overlapped.InternalHigh == 2);
		fd_table[fd]->overlapped.Internal = STATUS_PENDING;
		ResetEvent(fd_table[fd]->overlapped.hEvent);
		error = 0;
	}
	usbi_mutex_static_unlock(&fd_table_lock);

	if (error)
		goto err_out;

	return sizeof(unsigned char);

err_out:
	errno = error;
	return -1;
}
