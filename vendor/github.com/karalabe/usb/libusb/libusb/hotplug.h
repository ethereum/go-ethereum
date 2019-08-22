/* -*- Mode: C; indent-tabs-mode:t ; c-basic-offset:8 -*- */
/*
 * Hotplug support for libusb
 * Copyright © 2012-2013 Nathan Hjelm <hjelmn@mac.com>
 * Copyright © 2012-2013 Peter Stuge <peter@stuge.se>
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

#ifndef USBI_HOTPLUG_H
#define USBI_HOTPLUG_H

#include "libusbi.h"

enum usbi_hotplug_flags {
	/* This callback is interested in device arrivals */
	USBI_HOTPLUG_DEVICE_ARRIVED = LIBUSB_HOTPLUG_EVENT_DEVICE_ARRIVED,

	/* This callback is interested in device removals */
	USBI_HOTPLUG_DEVICE_LEFT = LIBUSB_HOTPLUG_EVENT_DEVICE_LEFT,

	/* IMPORTANT: The values for the below entries must start *after*
	 * the highest value of the above entries!!!
	 */

	/* The vendor_id field is valid for matching */
	USBI_HOTPLUG_VENDOR_ID_VALID = (1 << 3),

	/* The product_id field is valid for matching */
	USBI_HOTPLUG_PRODUCT_ID_VALID = (1 << 4),

	/* The dev_class field is valid for matching */
	USBI_HOTPLUG_DEV_CLASS_VALID = (1 << 5),

	/* This callback has been unregistered and needs to be freed */
	USBI_HOTPLUG_NEEDS_FREE = (1 << 6),
};

/** \ingroup hotplug
 * The hotplug callback structure. The user populates this structure with
 * libusb_hotplug_prepare_callback() and then calls libusb_hotplug_register_callback()
 * to receive notification of hotplug events.
 */
struct libusb_hotplug_callback {
	/** Flags that control how this callback behaves */
	uint8_t flags;

	/** Vendor ID to match (if flags says this is valid) */
	uint16_t vendor_id;

	/** Product ID to match (if flags says this is valid) */
	uint16_t product_id;

	/** Device class to match (if flags says this is valid) */
	uint8_t dev_class;

	/** Callback function to invoke for matching event/device */
	libusb_hotplug_callback_fn cb;

	/** Handle for this callback (used to match on deregister) */
	libusb_hotplug_callback_handle handle;

	/** User data that will be passed to the callback function */
	void *user_data;

	/** List this callback is registered in (ctx->hotplug_cbs) */
	struct list_head list;
};

struct libusb_hotplug_message {
	/** The hotplug event that occurred */
	libusb_hotplug_event event;

	/** The device for which this hotplug event occurred */
	struct libusb_device *device;

	/** List this message is contained in (ctx->hotplug_msgs) */
	struct list_head list;
};

void usbi_hotplug_deregister(struct libusb_context *ctx, int forced);
void usbi_hotplug_match(struct libusb_context *ctx, struct libusb_device *dev,
			libusb_hotplug_event event);
void usbi_hotplug_notification(struct libusb_context *ctx, struct libusb_device *dev,
			libusb_hotplug_event event);

#endif
