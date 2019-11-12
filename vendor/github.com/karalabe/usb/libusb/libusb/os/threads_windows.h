/*
 * libusb synchronization on Microsoft Windows
 *
 * Copyright © 2010 Michael Plante <michael.plante@gmail.com>
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
 */

#ifndef LIBUSB_THREADS_WINDOWS_H
#define LIBUSB_THREADS_WINDOWS_H

#define USBI_MUTEX_INITIALIZER	0L
#ifdef _WIN32_WCE
typedef LONG usbi_mutex_static_t;
#else
typedef volatile LONG usbi_mutex_static_t;
#endif
void usbi_mutex_static_lock(usbi_mutex_static_t *mutex);
static inline void usbi_mutex_static_unlock(usbi_mutex_static_t *mutex)
{
	InterlockedExchange(mutex, 0L);
}

typedef CRITICAL_SECTION usbi_mutex_t;
static inline int usbi_mutex_init(usbi_mutex_t *mutex)
{
	InitializeCriticalSection(mutex);
	return 0;
}
static inline void usbi_mutex_lock(usbi_mutex_t *mutex)
{
	EnterCriticalSection(mutex);
}
static inline void usbi_mutex_unlock(usbi_mutex_t *mutex)
{
	LeaveCriticalSection(mutex);
}
static inline int usbi_mutex_trylock(usbi_mutex_t *mutex)
{
	return !TryEnterCriticalSection(mutex);
}
static inline void usbi_mutex_destroy(usbi_mutex_t *mutex)
{
	DeleteCriticalSection(mutex);
}

// We *were* getting timespec from pthread.h:
#if (!defined(HAVE_STRUCT_TIMESPEC) && !defined(_TIMESPEC_DEFINED))
#define HAVE_STRUCT_TIMESPEC 1
#define _TIMESPEC_DEFINED 1
struct timespec {
	long tv_sec;
	long tv_nsec;
};
#endif /* HAVE_STRUCT_TIMESPEC | _TIMESPEC_DEFINED */

// We *were* getting ETIMEDOUT from pthread.h:
#ifndef ETIMEDOUT
#define ETIMEDOUT	10060	/* This is the value in winsock.h. */
#endif

typedef struct usbi_cond {
	// Every time a thread touches the CV, it winds up in one of these lists.
	//   It stays there until the CV is destroyed, even if the thread terminates.
	struct list_head waiters;
	struct list_head not_waiting;
} usbi_cond_t;

void usbi_cond_init(usbi_cond_t *cond);
int usbi_cond_wait(usbi_cond_t *cond, usbi_mutex_t *mutex);
int usbi_cond_timedwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, const struct timeval *tv);
void usbi_cond_broadcast(usbi_cond_t *cond);
void usbi_cond_destroy(usbi_cond_t *cond);

typedef DWORD usbi_tls_key_t;
static inline void usbi_tls_key_create(usbi_tls_key_t *key)
{
	*key = TlsAlloc();
}
static inline void *usbi_tls_key_get(usbi_tls_key_t key)
{
	return TlsGetValue(key);
}
static inline void usbi_tls_key_set(usbi_tls_key_t key, void *ptr)
{
	(void)TlsSetValue(key, ptr);
}
static inline void usbi_tls_key_delete(usbi_tls_key_t key)
{
	(void)TlsFree(key);
}

static inline int usbi_get_tid(void)
{
	return (int)GetCurrentThreadId();
}

#endif /* LIBUSB_THREADS_WINDOWS_H */
