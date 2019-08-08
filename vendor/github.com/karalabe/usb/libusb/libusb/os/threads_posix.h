/*
 * libusb synchronization using POSIX Threads
 *
 * Copyright © 2010 Peter Stuge <peter@stuge.se>
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

#ifndef LIBUSB_THREADS_POSIX_H
#define LIBUSB_THREADS_POSIX_H

#include <pthread.h>
#ifdef HAVE_SYS_TIME_H
#include <sys/time.h>
#endif

#define USBI_MUTEX_INITIALIZER	PTHREAD_MUTEX_INITIALIZER
typedef pthread_mutex_t usbi_mutex_static_t;
static inline void usbi_mutex_static_lock(usbi_mutex_static_t *mutex)
{
	(void)pthread_mutex_lock(mutex);
}
static inline void usbi_mutex_static_unlock(usbi_mutex_static_t *mutex)
{
	(void)pthread_mutex_unlock(mutex);
}

typedef pthread_mutex_t usbi_mutex_t;
static inline int usbi_mutex_init(usbi_mutex_t *mutex)
{
	return pthread_mutex_init(mutex, NULL);
}
static inline void usbi_mutex_lock(usbi_mutex_t *mutex)
{
	(void)pthread_mutex_lock(mutex);
}
static inline void usbi_mutex_unlock(usbi_mutex_t *mutex)
{
	(void)pthread_mutex_unlock(mutex);
}
static inline int usbi_mutex_trylock(usbi_mutex_t *mutex)
{
	return pthread_mutex_trylock(mutex);
}
static inline void usbi_mutex_destroy(usbi_mutex_t *mutex)
{
	(void)pthread_mutex_destroy(mutex);
}

typedef pthread_cond_t usbi_cond_t;
static inline void usbi_cond_init(pthread_cond_t *cond)
{
	(void)pthread_cond_init(cond, NULL);
}
static inline int usbi_cond_wait(usbi_cond_t *cond, usbi_mutex_t *mutex)
{
	return pthread_cond_wait(cond, mutex);
}
int usbi_cond_timedwait(usbi_cond_t *cond,
	usbi_mutex_t *mutex, const struct timeval *tv);
static inline void usbi_cond_broadcast(usbi_cond_t *cond)
{
	(void)pthread_cond_broadcast(cond);
}
static inline void usbi_cond_destroy(usbi_cond_t *cond)
{
	(void)pthread_cond_destroy(cond);
}

typedef pthread_key_t usbi_tls_key_t;
static inline void usbi_tls_key_create(usbi_tls_key_t *key)
{
	(void)pthread_key_create(key, NULL);
}
static inline void *usbi_tls_key_get(usbi_tls_key_t key)
{
	return pthread_getspecific(key);
}
static inline void usbi_tls_key_set(usbi_tls_key_t key, void *ptr)
{
	(void)pthread_setspecific(key, ptr);
}
static inline void usbi_tls_key_delete(usbi_tls_key_t key)
{
	(void)pthread_key_delete(key);
}

int usbi_get_tid(void);

#endif /* LIBUSB_THREADS_POSIX_H */
