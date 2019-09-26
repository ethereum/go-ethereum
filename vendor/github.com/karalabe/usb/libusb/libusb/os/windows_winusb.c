/*
 * windows backend for libusb 1.0
 * Copyright © 2009-2012 Pete Batard <pete@akeo.ie>
 * Copyright © 2016-2018 Chris Dickens <christopher.a.dickens@gmail.com>
 * With contributions from Michael Plante, Orin Eman et al.
 * Parts of this code adapted from libusb-win32-v1 by Stephan Meyer
 * HID Reports IOCTLs inspired from HIDAPI by Alan Ott, Signal 11 Software
 * Hash table functions adapted from glibc, by Ulrich Drepper et al.
 * Major code testing contribution by Xiaofan Chen
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

#include <config.h>

#include <windows.h>
#include <setupapi.h>
#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <process.h>
#include <stdio.h>
#include <inttypes.h>
#include <objbase.h>
#include <winioctl.h>

#include "libusbi.h"
#include "windows_common.h"
#include "windows_nt_common.h"
#include "windows_winusb.h"

// Unfuckup the 'inferface' keyword
#undef interface

#define HANDLE_VALID(h) (((h) != NULL) && ((h) != INVALID_HANDLE_VALUE))

// The 2 macros below are used in conjunction with safe loops.
#define LOOP_CHECK(fcall)			\
	{					\
		r = fcall;			\
		if (r != LIBUSB_SUCCESS)	\
			continue;		\
	}
#define LOOP_BREAK(err)				\
	{					\
		r = err;			\
		continue;			\
	}

// WinUSB-like API prototypes
static int winusbx_init(struct libusb_context *ctx);
static void winusbx_exit(void);
static int winusbx_open(int sub_api, struct libusb_device_handle *dev_handle);
static void winusbx_close(int sub_api, struct libusb_device_handle *dev_handle);
static int winusbx_configure_endpoints(int sub_api, struct libusb_device_handle *dev_handle, int iface);
static int winusbx_claim_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface);
static int winusbx_release_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface);
static int winusbx_submit_control_transfer(int sub_api, struct usbi_transfer *itransfer);
static int winusbx_set_interface_altsetting(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting);
static int winusbx_submit_iso_transfer(int sub_api, struct usbi_transfer *itransfer);
static int winusbx_submit_bulk_transfer(int sub_api, struct usbi_transfer *itransfer);
static int winusbx_clear_halt(int sub_api, struct libusb_device_handle *dev_handle, unsigned char endpoint);
static int winusbx_abort_transfers(int sub_api, struct usbi_transfer *itransfer);
static int winusbx_abort_control(int sub_api, struct usbi_transfer *itransfer);
static int winusbx_reset_device(int sub_api, struct libusb_device_handle *dev_handle);
static int winusbx_copy_transfer_data(int sub_api, struct usbi_transfer *itransfer, uint32_t io_size);
// Composite API prototypes
static int composite_open(int sub_api, struct libusb_device_handle *dev_handle);
static void composite_close(int sub_api, struct libusb_device_handle *dev_handle);
static int composite_claim_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface);
static int composite_set_interface_altsetting(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting);
static int composite_release_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface);
static int composite_submit_control_transfer(int sub_api, struct usbi_transfer *itransfer);
static int composite_submit_bulk_transfer(int sub_api, struct usbi_transfer *itransfer);
static int composite_submit_iso_transfer(int sub_api, struct usbi_transfer *itransfer);
static int composite_clear_halt(int sub_api, struct libusb_device_handle *dev_handle, unsigned char endpoint);
static int composite_abort_transfers(int sub_api, struct usbi_transfer *itransfer);
static int composite_abort_control(int sub_api, struct usbi_transfer *itransfer);
static int composite_reset_device(int sub_api, struct libusb_device_handle *dev_handle);
static int composite_copy_transfer_data(int sub_api, struct usbi_transfer *itransfer, uint32_t io_size);

static usbi_mutex_t autoclaim_lock;

// API globals
static HMODULE WinUSBX_handle = NULL;
static struct winusb_interface WinUSBX[SUB_API_MAX];
#define CHECK_WINUSBX_AVAILABLE(sub_api)		\
	do {						\
		if (sub_api == SUB_API_NOTSET)		\
			sub_api = priv->sub_api;	\
		if (!WinUSBX[sub_api].initialized) 	\
			return LIBUSB_ERROR_ACCESS;	\
	} while (0)

static bool api_hid_available = false;
#define CHECK_HID_AVAILABLE				\
	do {						\
		if (!api_hid_available)			\
			return LIBUSB_ERROR_ACCESS;	\
	} while (0)

#if defined(ENABLE_LOGGING)
static const char *guid_to_string(const GUID *guid)
{
	static char guid_string[MAX_GUID_STRING_LENGTH];

	if (guid == NULL)
		return "";

	sprintf(guid_string, "{%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}",
		(unsigned int)guid->Data1, guid->Data2, guid->Data3,
		guid->Data4[0], guid->Data4[1], guid->Data4[2], guid->Data4[3],
		guid->Data4[4], guid->Data4[5], guid->Data4[6], guid->Data4[7]);

	return guid_string;
}
#endif

/*
 * Sanitize Microsoft's paths: convert to uppercase, add prefix and fix backslashes.
 * Return an allocated sanitized string or NULL on error.
 */
static char *sanitize_path(const char *path)
{
	const char root_prefix[] = {'\\', '\\', '.', '\\'};
	size_t j, size;
	char *ret_path;
	size_t add_root = 0;

	if (path == NULL)
		return NULL;

	size = strlen(path) + 1;

	// Microsoft indiscriminately uses '\\?\', '\\.\', '##?#" or "##.#" for root prefixes.
	if (!((size > 3) && (((path[0] == '\\') && (path[1] == '\\') && (path[3] == '\\'))
			|| ((path[0] == '#') && (path[1] == '#') && (path[3] == '#'))))) {
		add_root = sizeof(root_prefix);
		size += add_root;
	}

	ret_path = malloc(size);
	if (ret_path == NULL)
		return NULL;

	strcpy(&ret_path[add_root], path);

	// Ensure consistency with root prefix
	memcpy(ret_path, root_prefix, sizeof(root_prefix));

	// Same goes for '\' and '#' after the root prefix. Ensure '#' is used
	for (j = sizeof(root_prefix); j < size; j++) {
		ret_path[j] = (char)toupper((int)ret_path[j]); // Fix case too
		if (ret_path[j] == '\\')
			ret_path[j] = '#';
	}

	return ret_path;
}

/*
 * Cfgmgr32, AdvAPI32, OLE32 and SetupAPI DLL functions
 */
static BOOL init_dlls(void)
{
	DLL_GET_HANDLE(Cfgmgr32);
	DLL_LOAD_FUNC(Cfgmgr32, CM_Get_Parent, TRUE);
	DLL_LOAD_FUNC(Cfgmgr32, CM_Get_Child, TRUE);

	// Prefixed to avoid conflict with header files
	DLL_GET_HANDLE(AdvAPI32);
	DLL_LOAD_FUNC_PREFIXED(AdvAPI32, p, RegQueryValueExW, TRUE);
	DLL_LOAD_FUNC_PREFIXED(AdvAPI32, p, RegCloseKey, TRUE);

	DLL_GET_HANDLE(OLE32);
	DLL_LOAD_FUNC_PREFIXED(OLE32, p, IIDFromString, TRUE);

	DLL_GET_HANDLE(SetupAPI);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiGetClassDevsA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiEnumDeviceInfo, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiEnumDeviceInterfaces, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiGetDeviceInstanceIdA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiGetDeviceInterfaceDetailA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiGetDeviceRegistryPropertyA, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiDestroyDeviceInfoList, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiOpenDevRegKey, TRUE);
	DLL_LOAD_FUNC_PREFIXED(SetupAPI, p, SetupDiOpenDeviceInterfaceRegKey, TRUE);

	return TRUE;
}

static void exit_dlls(void)
{
	DLL_FREE_HANDLE(Cfgmgr32);
	DLL_FREE_HANDLE(AdvAPI32);
	DLL_FREE_HANDLE(OLE32);
	DLL_FREE_HANDLE(SetupAPI);
}

/*
 * enumerate interfaces for the whole USB class
 *
 * Parameters:
 * dev_info: a pointer to a dev_info list
 * dev_info_data: a pointer to an SP_DEVINFO_DATA to be filled (or NULL if not needed)
 * enumerator: the generic USB class for which to retrieve interface details
 * index: zero based index of the interface in the device info list
 *
 * Note: it is the responsibility of the caller to free the DEVICE_INTERFACE_DETAIL_DATA
 * structure returned and call this function repeatedly using the same guid (with an
 * incremented index starting at zero) until all interfaces have been returned.
 */
static bool get_devinfo_data(struct libusb_context *ctx,
	HDEVINFO *dev_info, SP_DEVINFO_DATA *dev_info_data, const char *enumerator, unsigned _index)
{
	if (_index == 0) {
		*dev_info = pSetupDiGetClassDevsA(NULL, enumerator, NULL, DIGCF_PRESENT|DIGCF_ALLCLASSES);
		if (*dev_info == INVALID_HANDLE_VALUE) {
			usbi_err(ctx, "could not obtain device info set for PnP enumerator '%s': %s",
				enumerator, windows_error_str(0));
			return false;
		}
	}

	dev_info_data->cbSize = sizeof(SP_DEVINFO_DATA);
	if (!pSetupDiEnumDeviceInfo(*dev_info, _index, dev_info_data)) {
		if (GetLastError() != ERROR_NO_MORE_ITEMS)
			usbi_err(ctx, "could not obtain device info data for PnP enumerator '%s' index %u: %s",
				enumerator, _index, windows_error_str(0));

		pSetupDiDestroyDeviceInfoList(*dev_info);
		*dev_info = INVALID_HANDLE_VALUE;
		return false;
	}
	return true;
}

/*
 * enumerate interfaces for a specific GUID
 *
 * Parameters:
 * dev_info: a pointer to a dev_info list
 * dev_info_data: a pointer to an SP_DEVINFO_DATA to be filled (or NULL if not needed)
 * guid: the GUID for which to retrieve interface details
 * index: zero based index of the interface in the device info list
 *
 * Note: it is the responsibility of the caller to free the DEVICE_INTERFACE_DETAIL_DATA
 * structure returned and call this function repeatedly using the same guid (with an
 * incremented index starting at zero) until all interfaces have been returned.
 */
static int get_interface_details(struct libusb_context *ctx, HDEVINFO dev_info,
	PSP_DEVINFO_DATA dev_info_data, LPCGUID guid, DWORD *_index, char **dev_interface_path)
{
	SP_DEVICE_INTERFACE_DATA dev_interface_data;
	PSP_DEVICE_INTERFACE_DETAIL_DATA_A dev_interface_details;
	DWORD size;

	dev_info_data->cbSize = sizeof(SP_DEVINFO_DATA);
	dev_interface_data.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);
	for (;;) {
		if (!pSetupDiEnumDeviceInfo(dev_info, *_index, dev_info_data)) {
			if (GetLastError() != ERROR_NO_MORE_ITEMS) {
				usbi_err(ctx, "Could not obtain device info data for %s index %u: %s",
					guid_to_string(guid), *_index, windows_error_str(0));
				return LIBUSB_ERROR_OTHER;
			}

			// No more devices
			return LIBUSB_SUCCESS;
		}

		// Always advance the index for the next iteration
		(*_index)++;

		if (pSetupDiEnumDeviceInterfaces(dev_info, dev_info_data, guid, 0, &dev_interface_data))
			break;

		if (GetLastError() != ERROR_NO_MORE_ITEMS) {
			usbi_err(ctx, "Could not obtain interface data for %s devInst %X: %s",
				guid_to_string(guid), dev_info_data->DevInst, windows_error_str(0));
			return LIBUSB_ERROR_OTHER;
		}

		// Device does not have an interface matching this GUID, skip
	}

	// Read interface data (dummy + actual) to access the device path
	if (!pSetupDiGetDeviceInterfaceDetailA(dev_info, &dev_interface_data, NULL, 0, &size, NULL)) {
		// The dummy call should fail with ERROR_INSUFFICIENT_BUFFER
		if (GetLastError() != ERROR_INSUFFICIENT_BUFFER) {
			usbi_err(ctx, "could not access interface data (dummy) for %s devInst %X: %s",
				guid_to_string(guid), dev_info_data->DevInst, windows_error_str(0));
			return LIBUSB_ERROR_OTHER;
		}
	} else {
		usbi_err(ctx, "program assertion failed - http://msdn.microsoft.com/en-us/library/ms792901.aspx is wrong");
		return LIBUSB_ERROR_OTHER;
	}

	dev_interface_details = malloc(size);
	if (dev_interface_details == NULL) {
		usbi_err(ctx, "could not allocate interface data for %s devInst %X",
			guid_to_string(guid), dev_info_data->DevInst);
		return LIBUSB_ERROR_NO_MEM;
	}

	dev_interface_details->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_A);
	if (!pSetupDiGetDeviceInterfaceDetailA(dev_info, &dev_interface_data,
		dev_interface_details, size, NULL, NULL)) {
		usbi_err(ctx, "could not access interface data (actual) for %s devInst %X: %s",
			guid_to_string(guid), dev_info_data->DevInst, windows_error_str(0));
		free(dev_interface_details);
		return LIBUSB_ERROR_OTHER;
	}

	*dev_interface_path = sanitize_path(dev_interface_details->DevicePath);
	free(dev_interface_details);

	if (*dev_interface_path == NULL) {
		usbi_err(ctx, "could not allocate interface path for %s devInst %X",
			guid_to_string(guid), dev_info_data->DevInst);
		return LIBUSB_ERROR_NO_MEM;
	}

	return LIBUSB_SUCCESS;
}

/* For libusb0 filter */
static SP_DEVICE_INTERFACE_DETAIL_DATA_A *get_interface_details_filter(struct libusb_context *ctx,
	HDEVINFO *dev_info, SP_DEVINFO_DATA *dev_info_data, const GUID *guid, unsigned _index, char *filter_path)
{
	SP_DEVICE_INTERFACE_DATA dev_interface_data;
	SP_DEVICE_INTERFACE_DETAIL_DATA_A *dev_interface_details;
	DWORD size;

	if (_index == 0)
		*dev_info = pSetupDiGetClassDevsA(guid, NULL, NULL, DIGCF_PRESENT|DIGCF_DEVICEINTERFACE);

	if (dev_info_data != NULL) {
		dev_info_data->cbSize = sizeof(SP_DEVINFO_DATA);
		if (!pSetupDiEnumDeviceInfo(*dev_info, _index, dev_info_data)) {
			if (GetLastError() != ERROR_NO_MORE_ITEMS)
				usbi_err(ctx, "Could not obtain device info data for index %u: %s",
					_index, windows_error_str(0));

			pSetupDiDestroyDeviceInfoList(*dev_info);
			*dev_info = INVALID_HANDLE_VALUE;
			return NULL;
		}
	}

	dev_interface_data.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);
	if (!pSetupDiEnumDeviceInterfaces(*dev_info, NULL, guid, _index, &dev_interface_data)) {
		if (GetLastError() != ERROR_NO_MORE_ITEMS)
			usbi_err(ctx, "Could not obtain interface data for index %u: %s",
				_index, windows_error_str(0));

		pSetupDiDestroyDeviceInfoList(*dev_info);
		*dev_info = INVALID_HANDLE_VALUE;
		return NULL;
	}

	// Read interface data (dummy + actual) to access the device path
	if (!pSetupDiGetDeviceInterfaceDetailA(*dev_info, &dev_interface_data, NULL, 0, &size, NULL)) {
		// The dummy call should fail with ERROR_INSUFFICIENT_BUFFER
		if (GetLastError() != ERROR_INSUFFICIENT_BUFFER) {
			usbi_err(ctx, "could not access interface data (dummy) for index %u: %s",
				_index, windows_error_str(0));
			goto err_exit;
		}
	} else {
		usbi_err(ctx, "program assertion failed - http://msdn.microsoft.com/en-us/library/ms792901.aspx is wrong.");
		goto err_exit;
	}

	dev_interface_details = calloc(1, size);
	if (dev_interface_details == NULL) {
		usbi_err(ctx, "could not allocate interface data for index %u.", _index);
		goto err_exit;
	}

	dev_interface_details->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_A);
	if (!pSetupDiGetDeviceInterfaceDetailA(*dev_info, &dev_interface_data, dev_interface_details, size, &size, NULL))
		usbi_err(ctx, "could not access interface data (actual) for index %u: %s",
			_index, windows_error_str(0));

	// [trobinso] lookup the libusb0 symbolic index.
	if (dev_interface_details) {
		HKEY hkey_device_interface = pSetupDiOpenDeviceInterfaceRegKey(*dev_info, &dev_interface_data, 0, KEY_READ);
		if (hkey_device_interface != INVALID_HANDLE_VALUE) {
			DWORD libusb0_symboliclink_index = 0;
			DWORD value_length = sizeof(DWORD);
			DWORD value_type = 0;
			LONG status;

			status = pRegQueryValueExW(hkey_device_interface, L"LUsb0", NULL, &value_type,
				(LPBYTE)&libusb0_symboliclink_index, &value_length);
			if (status == ERROR_SUCCESS) {
				if (libusb0_symboliclink_index < 256) {
					// libusb0.sys is connected to this device instance.
					// If the the device interface guid is {F9F3FF14-AE21-48A0-8A25-8011A7A931D9} then it's a filter.
					sprintf(filter_path, "\\\\.\\libusb0-%04u", (unsigned int)libusb0_symboliclink_index);
					usbi_dbg("assigned libusb0 symbolic link %s", filter_path);
				} else {
					// libusb0.sys was connected to this device instance at one time; but not anymore.
				}
			}
			pRegCloseKey(hkey_device_interface);
		}
	}

	return dev_interface_details;

err_exit:
	pSetupDiDestroyDeviceInfoList(*dev_info);
	*dev_info = INVALID_HANDLE_VALUE;
	return NULL;
}

/*
 * Returns the first known ancestor of a device
 */
static struct libusb_device *get_ancestor(struct libusb_context *ctx,
	DEVINST devinst, PDEVINST _parent_devinst)
{
	struct libusb_device *dev = NULL;
	DEVINST parent_devinst;

	while (dev == NULL) {
		if (CM_Get_Parent(&parent_devinst, devinst, 0) != CR_SUCCESS)
			break;
		devinst = parent_devinst;
		dev = usbi_get_device_by_session_id(ctx, (unsigned long)devinst);
	}

	if ((dev != NULL) && (_parent_devinst != NULL))
		*_parent_devinst = devinst;

	return dev;
}

/*
 * Determine which interface the given endpoint address belongs to
 */
static int get_interface_by_endpoint(struct libusb_config_descriptor *conf_desc, uint8_t ep)
{
	const struct libusb_interface *intf;
	const struct libusb_interface_descriptor *intf_desc;
	int i, j, k;

	for (i = 0; i < conf_desc->bNumInterfaces; i++) {
		intf = &conf_desc->interface[i];
		for (j = 0; j < intf->num_altsetting; j++) {
			intf_desc = &intf->altsetting[j];
			for (k = 0; k < intf_desc->bNumEndpoints; k++) {
				if (intf_desc->endpoint[k].bEndpointAddress == ep) {
					usbi_dbg("found endpoint %02X on interface %d", intf_desc->bInterfaceNumber, i);
					return intf_desc->bInterfaceNumber;
				}
			}
		}
	}

	usbi_dbg("endpoint %02X not found on any interface", ep);
	return LIBUSB_ERROR_NOT_FOUND;
}

/*
 * Populate the endpoints addresses of the device_priv interface helper structs
 */
static int windows_assign_endpoints(struct libusb_device_handle *dev_handle, int iface, int altsetting)
{
	int i, r;
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	struct libusb_config_descriptor *conf_desc;
	const struct libusb_interface_descriptor *if_desc;
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);

	r = libusb_get_active_config_descriptor(dev_handle->dev, &conf_desc);
	if (r != LIBUSB_SUCCESS) {
		usbi_warn(ctx, "could not read config descriptor: error %d", r);
		return r;
	}

	if_desc = &conf_desc->interface[iface].altsetting[altsetting];
	safe_free(priv->usb_interface[iface].endpoint);

	if (if_desc->bNumEndpoints == 0) {
		usbi_dbg("no endpoints found for interface %d", iface);
		libusb_free_config_descriptor(conf_desc);
		return LIBUSB_SUCCESS;
	}

	priv->usb_interface[iface].endpoint = malloc(if_desc->bNumEndpoints);
	if (priv->usb_interface[iface].endpoint == NULL) {
		libusb_free_config_descriptor(conf_desc);
		return LIBUSB_ERROR_NO_MEM;
	}

	priv->usb_interface[iface].nb_endpoints = if_desc->bNumEndpoints;
	for (i = 0; i < if_desc->bNumEndpoints; i++) {
		priv->usb_interface[iface].endpoint[i] = if_desc->endpoint[i].bEndpointAddress;
		usbi_dbg("(re)assigned endpoint %02X to interface %d", priv->usb_interface[iface].endpoint[i], iface);
	}
	libusb_free_config_descriptor(conf_desc);

	// Extra init may be required to configure endpoints
	if (priv->apib->configure_endpoints)
		r = priv->apib->configure_endpoints(SUB_API_NOTSET, dev_handle, iface);

	return r;
}

// Lookup for a match in the list of API driver names
// return -1 if not found, driver match number otherwise
static int get_sub_api(char *driver, int api)
{
	int i;
	const char sep_str[2] = {LIST_SEPARATOR, 0};
	char *tok, *tmp_str;
	size_t len = strlen(driver);

	if (len == 0)
		return SUB_API_NOTSET;

	tmp_str = _strdup(driver);
	if (tmp_str == NULL)
		return SUB_API_NOTSET;

	tok = strtok(tmp_str, sep_str);
	while (tok != NULL) {
		for (i = 0; i < usb_api_backend[api].nb_driver_names; i++) {
			if (_stricmp(tok, usb_api_backend[api].driver_name_list[i]) == 0) {
				free(tmp_str);
				return i;
			}
		}
		tok = strtok(NULL, sep_str);
	}

	free(tmp_str);
	return SUB_API_NOTSET;
}

/*
 * auto-claiming and auto-release helper functions
 */
static int auto_claim(struct libusb_transfer *transfer, int *interface_number, int api_type)
{
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(
		transfer->dev_handle);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface = *interface_number;
	int r = LIBUSB_SUCCESS;

	switch (api_type) {
	case USB_API_WINUSBX:
	case USB_API_HID:
		break;
	default:
		return LIBUSB_ERROR_INVALID_PARAM;
	}

	usbi_mutex_lock(&autoclaim_lock);
	if (current_interface < 0) { // No serviceable interface was found
		for (current_interface = 0; current_interface < USB_MAXINTERFACES; current_interface++) {
			// Must claim an interface of the same API type
			if ((priv->usb_interface[current_interface].apib->id == api_type)
					&& (libusb_claim_interface(transfer->dev_handle, current_interface) == LIBUSB_SUCCESS)) {
				usbi_dbg("auto-claimed interface %d for control request", current_interface);
				if (handle_priv->autoclaim_count[current_interface] != 0)
					usbi_warn(ctx, "program assertion failed - autoclaim_count was nonzero");
				handle_priv->autoclaim_count[current_interface]++;
				break;
			}
		}
		if (current_interface == USB_MAXINTERFACES) {
			usbi_err(ctx, "could not auto-claim any interface");
			r = LIBUSB_ERROR_NOT_FOUND;
		}
	} else {
		// If we have a valid interface that was autoclaimed, we must increment
		// its autoclaim count so that we can prevent an early release.
		if (handle_priv->autoclaim_count[current_interface] != 0)
			handle_priv->autoclaim_count[current_interface]++;
	}
	usbi_mutex_unlock(&autoclaim_lock);

	*interface_number = current_interface;
	return r;
}

static void auto_release(struct usbi_transfer *itransfer)
{
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	libusb_device_handle *dev_handle = transfer->dev_handle;
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	int r;

	usbi_mutex_lock(&autoclaim_lock);
	if (handle_priv->autoclaim_count[transfer_priv->interface_number] > 0) {
		handle_priv->autoclaim_count[transfer_priv->interface_number]--;
		if (handle_priv->autoclaim_count[transfer_priv->interface_number] == 0) {
			r = libusb_release_interface(dev_handle, transfer_priv->interface_number);
			if (r == LIBUSB_SUCCESS)
				usbi_dbg("auto-released interface %d", transfer_priv->interface_number);
			else
				usbi_dbg("failed to auto-release interface %d (%s)",
					transfer_priv->interface_number, libusb_error_name((enum libusb_error)r));
		}
	}
	usbi_mutex_unlock(&autoclaim_lock);
}

/*
 * init: libusb backend init function
 */
static int winusb_init(struct libusb_context *ctx)
{
	int i;

	// We need a lock for proper auto-release
	usbi_mutex_init(&autoclaim_lock);

	// Load DLL imports
	if (!init_dlls()) {
		usbi_err(ctx, "could not resolve DLL functions");
		return LIBUSB_ERROR_OTHER;
	}

	// Initialize the low level APIs (we don't care about errors at this stage)
	for (i = 0; i < USB_API_MAX; i++) {
		if (usb_api_backend[i].init && usb_api_backend[i].init(ctx))
			usbi_warn(ctx, "error initializing %s backend",
				usb_api_backend[i].designation);
	}

	return LIBUSB_SUCCESS;
}

/*
* exit: libusb backend deinitialization function
*/
static void winusb_exit(struct libusb_context *ctx)
{
	int i;

	for (i = 0; i < USB_API_MAX; i++) {
		if (usb_api_backend[i].exit)
			usb_api_backend[i].exit();
	}

	exit_dlls();
	usbi_mutex_destroy(&autoclaim_lock);
}

/*
 * fetch and cache all the config descriptors through I/O
 */
static void cache_config_descriptors(struct libusb_device *dev, HANDLE hub_handle)
{
	struct libusb_context *ctx = DEVICE_CTX(dev);
	struct winusb_device_priv *priv = _device_priv(dev);
	DWORD size, ret_size;
	uint8_t i;

	USB_CONFIGURATION_DESCRIPTOR_SHORT cd_buf_short; // dummy request
	PUSB_DESCRIPTOR_REQUEST cd_buf_actual = NULL;    // actual request
	PUSB_CONFIGURATION_DESCRIPTOR cd_data;

	if (dev->num_configurations == 0)
		return;

	priv->config_descriptor = calloc(dev->num_configurations, sizeof(PUSB_CONFIGURATION_DESCRIPTOR));
	if (priv->config_descriptor == NULL) {
		usbi_err(ctx, "could not allocate configuration descriptor array for '%s'", priv->dev_id);
		return;
	}

	for (i = 0; i <= dev->num_configurations; i++) {
		safe_free(cd_buf_actual);

		if (i == dev->num_configurations)
			break;

		size = sizeof(cd_buf_short);
		memset(&cd_buf_short, 0, size);

		cd_buf_short.req.ConnectionIndex = (ULONG)dev->port_number;
		cd_buf_short.req.SetupPacket.bmRequest = LIBUSB_ENDPOINT_IN;
		cd_buf_short.req.SetupPacket.bRequest = LIBUSB_REQUEST_GET_DESCRIPTOR;
		cd_buf_short.req.SetupPacket.wValue = (LIBUSB_DT_CONFIG << 8) | i;
		cd_buf_short.req.SetupPacket.wIndex = 0;
		cd_buf_short.req.SetupPacket.wLength = (USHORT)sizeof(USB_CONFIGURATION_DESCRIPTOR);

		// Dummy call to get the required data size. Initial failures are reported as info rather
		// than error as they can occur for non-penalizing situations, such as with some hubs.
		// coverity[tainted_data_argument]
		if (!DeviceIoControl(hub_handle, IOCTL_USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION, &cd_buf_short, size,
			&cd_buf_short, size, &ret_size, NULL)) {
			usbi_info(ctx, "could not access configuration descriptor %u (dummy) for '%s': %s", i, priv->dev_id, windows_error_str(0));
			continue;
		}

		if ((ret_size != size) || (cd_buf_short.desc.wTotalLength < sizeof(USB_CONFIGURATION_DESCRIPTOR))) {
			usbi_info(ctx, "unexpected configuration descriptor %u size (dummy) for '%s'", i, priv->dev_id);
			continue;
		}

		size = sizeof(USB_DESCRIPTOR_REQUEST) + cd_buf_short.desc.wTotalLength;
		cd_buf_actual = malloc(size);
		if (cd_buf_actual == NULL) {
			usbi_err(ctx, "could not allocate configuration descriptor %u buffer for '%s'", i, priv->dev_id);
			continue;
		}

		// Actual call
		cd_buf_actual->ConnectionIndex = (ULONG)dev->port_number;
		cd_buf_actual->SetupPacket.bmRequest = LIBUSB_ENDPOINT_IN;
		cd_buf_actual->SetupPacket.bRequest = LIBUSB_REQUEST_GET_DESCRIPTOR;
		cd_buf_actual->SetupPacket.wValue = (LIBUSB_DT_CONFIG << 8) | i;
		cd_buf_actual->SetupPacket.wIndex = 0;
		cd_buf_actual->SetupPacket.wLength = cd_buf_short.desc.wTotalLength;

		if (!DeviceIoControl(hub_handle, IOCTL_USB_GET_DESCRIPTOR_FROM_NODE_CONNECTION, cd_buf_actual, size,
			cd_buf_actual, size, &ret_size, NULL)) {
			usbi_err(ctx, "could not access configuration descriptor %u (actual) for '%s': %s", i, priv->dev_id, windows_error_str(0));
			continue;
		}

		cd_data = (PUSB_CONFIGURATION_DESCRIPTOR)((UCHAR *)cd_buf_actual + sizeof(USB_DESCRIPTOR_REQUEST));

		if ((size != ret_size) || (cd_data->wTotalLength != cd_buf_short.desc.wTotalLength)) {
			usbi_err(ctx, "unexpected configuration descriptor %u size (actual) for '%s'", i, priv->dev_id);
			continue;
		}

		if (cd_data->bDescriptorType != LIBUSB_DT_CONFIG) {
			usbi_err(ctx, "descriptor %u not a configuration descriptor for '%s'", i, priv->dev_id);
			continue;
		}

		usbi_dbg("cached config descriptor %u (bConfigurationValue=%u, %u bytes)",
			i, cd_data->bConfigurationValue, cd_data->wTotalLength);

		// Cache the descriptor
		priv->config_descriptor[i] = malloc(cd_data->wTotalLength);
		if (priv->config_descriptor[i] != NULL) {
			memcpy(priv->config_descriptor[i], cd_data, cd_data->wTotalLength);
		} else {
			usbi_err(ctx, "could not allocate configuration descriptor %u buffer for '%s'", i, priv->dev_id);
		}
	}
}

/*
 * Populate a libusb device structure
 */
static int init_device(struct libusb_device *dev, struct libusb_device *parent_dev,
	uint8_t port_number, DEVINST devinst)
{
	struct libusb_context *ctx;
	struct libusb_device *tmp_dev;
	struct winusb_device_priv *priv, *parent_priv;
	USB_NODE_CONNECTION_INFORMATION_EX conn_info;
	USB_NODE_CONNECTION_INFORMATION_EX_V2 conn_info_v2;
	HANDLE hub_handle;
	DWORD size;
	uint8_t bus_number, depth;
	int r;

	priv = _device_priv(dev);

	// If the device is already initialized, we can stop here
	if (priv->initialized)
		return LIBUSB_SUCCESS;

	if (parent_dev != NULL) { // Not a HCD root hub
		ctx = DEVICE_CTX(dev);
		parent_priv = _device_priv(parent_dev);
		if (parent_priv->apib->id != USB_API_HUB) {
			usbi_warn(ctx, "parent for device '%s' is not a hub", priv->dev_id);
			return LIBUSB_ERROR_NOT_FOUND;
		}

		// Calculate depth and fetch bus number
		bus_number = parent_dev->bus_number;
		if (bus_number == 0) {
			tmp_dev = get_ancestor(ctx, devinst, &devinst);
			if (tmp_dev != parent_dev) {
				usbi_err(ctx, "program assertion failed - first ancestor is not parent");
				return LIBUSB_ERROR_NOT_FOUND;
			}
			libusb_unref_device(tmp_dev);

			for (depth = 1; bus_number == 0; depth++) {
				tmp_dev = get_ancestor(ctx, devinst, &devinst);
				if (tmp_dev->bus_number != 0) {
					bus_number = tmp_dev->bus_number;
					depth += _device_priv(tmp_dev)->depth;
				}
				libusb_unref_device(tmp_dev);
			}
		} else {
			depth = parent_priv->depth + 1;
		}

		if (bus_number == 0) {
			usbi_err(ctx, "program assertion failed - bus number not found for '%s'", priv->dev_id);
			return LIBUSB_ERROR_NOT_FOUND;
		}

		dev->bus_number = bus_number;
		dev->port_number = port_number;
		dev->parent_dev = parent_dev;
		priv->depth = depth;

		hub_handle = CreateFileA(parent_priv->path, GENERIC_WRITE, FILE_SHARE_WRITE, NULL, OPEN_EXISTING,
				     0, NULL);
		if (hub_handle == INVALID_HANDLE_VALUE) {
			usbi_warn(ctx, "could not open hub %s: %s", parent_priv->path, windows_error_str(0));
			return LIBUSB_ERROR_ACCESS;
		}

		memset(&conn_info, 0, sizeof(conn_info));
		conn_info.ConnectionIndex = (ULONG)port_number;
		// coverity[tainted_data_argument]
		if (!DeviceIoControl(hub_handle, IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX, &conn_info, sizeof(conn_info),
			&conn_info, sizeof(conn_info), &size, NULL)) {
			usbi_warn(ctx, "could not get node connection information for device '%s': %s",
				  priv->dev_id, windows_error_str(0));
			CloseHandle(hub_handle);
			return LIBUSB_ERROR_NO_DEVICE;
		}

		if (conn_info.ConnectionStatus == NoDeviceConnected) {
			usbi_err(ctx, "device '%s' is no longer connected!", priv->dev_id);
			CloseHandle(hub_handle);
			return LIBUSB_ERROR_NO_DEVICE;
		}

		memcpy(&priv->dev_descriptor, &(conn_info.DeviceDescriptor), sizeof(USB_DEVICE_DESCRIPTOR));
		dev->num_configurations = priv->dev_descriptor.bNumConfigurations;
		priv->active_config = conn_info.CurrentConfigurationValue;
		usbi_dbg("found %u configurations (active conf: %u)", dev->num_configurations, priv->active_config);

		// Cache as many config descriptors as we can
		cache_config_descriptors(dev, hub_handle);

		// In their great wisdom, Microsoft decided to BREAK the USB speed report between Windows 7 and Windows 8
		if (windows_version >= WINDOWS_8) {
			conn_info_v2.ConnectionIndex = (ULONG)port_number;
			conn_info_v2.Length = sizeof(USB_NODE_CONNECTION_INFORMATION_EX_V2);
			conn_info_v2.SupportedUsbProtocols.Usb300 = 1;
			if (!DeviceIoControl(hub_handle, IOCTL_USB_GET_NODE_CONNECTION_INFORMATION_EX_V2,
				&conn_info_v2, sizeof(conn_info_v2), &conn_info_v2, sizeof(conn_info_v2), &size, NULL)) {
				usbi_warn(ctx, "could not get node connection information (V2) for device '%s': %s",
					  priv->dev_id,  windows_error_str(0));
			} else if (conn_info_v2.Flags.DeviceIsOperatingAtSuperSpeedOrHigher) {
				conn_info.Speed = 3;
			}
		}

		CloseHandle(hub_handle);

		if (conn_info.DeviceAddress > UINT8_MAX)
			usbi_err(ctx, "program assertion failed - device address overflow");

		dev->device_address = (uint8_t)conn_info.DeviceAddress;

		switch (conn_info.Speed) {
		case 0: dev->speed = LIBUSB_SPEED_LOW; break;
		case 1: dev->speed = LIBUSB_SPEED_FULL; break;
		case 2: dev->speed = LIBUSB_SPEED_HIGH; break;
		case 3: dev->speed = LIBUSB_SPEED_SUPER; break;
		default:
			usbi_warn(ctx, "unknown device speed %u", conn_info.Speed);
			break;
		}
	}

	r = usbi_sanitize_device(dev);
	if (r)
		return r;

	priv->initialized = true;

	usbi_dbg("(bus: %u, addr: %u, depth: %u, port: %u): '%s'",
		dev->bus_number, dev->device_address, priv->depth, dev->port_number, priv->dev_id);

	return LIBUSB_SUCCESS;
}

static int enumerate_hcd_root_hub(struct libusb_context *ctx, const char *dev_id,
	uint8_t bus_number, DEVINST devinst)
{
	struct libusb_device *dev;
	struct winusb_device_priv *priv;
	unsigned long session_id;
	DEVINST child_devinst;

	if (CM_Get_Child(&child_devinst, devinst, 0) != CR_SUCCESS) {
		usbi_err(ctx, "could not get child devinst for '%s'", dev_id);
		return LIBUSB_ERROR_OTHER;
	}

	session_id = (unsigned long)child_devinst;
	dev = usbi_get_device_by_session_id(ctx, session_id);
	if (dev == NULL) {
		usbi_err(ctx, "program assertion failed - HCD '%s' child not found", dev_id);
		return LIBUSB_ERROR_NO_DEVICE;
	}

	if (dev->bus_number == 0) {
		// Only do this once
		usbi_dbg("assigning HCD '%s' bus number %u", dev_id, bus_number);
		priv = _device_priv(dev);
		dev->bus_number = bus_number;
		dev->num_configurations = 1;
		priv->dev_descriptor.bLength = LIBUSB_DT_DEVICE_SIZE;
		priv->dev_descriptor.bDescriptorType = LIBUSB_DT_DEVICE;
		priv->dev_descriptor.bDeviceClass = LIBUSB_CLASS_HUB;
		priv->dev_descriptor.bNumConfigurations = 1;
		priv->active_config = 1;
		priv->root_hub = true;
		if (sscanf(dev_id, "PCI\\VEN_%04hx&DEV_%04hx%*s", &priv->dev_descriptor.idVendor, &priv->dev_descriptor.idProduct) != 2) {
			usbi_warn(ctx, "could not infer VID/PID of HCD root hub from '%s'", dev_id);
			priv->dev_descriptor.idVendor = 0x1d6b; // Linux Foundation root hub
			priv->dev_descriptor.idProduct = 1;
		}
	}

	libusb_unref_device(dev);
	return LIBUSB_SUCCESS;
}

// Returns the api type, or 0 if not found/unsupported
static void get_api_type(struct libusb_context *ctx, HDEVINFO *dev_info,
	SP_DEVINFO_DATA *dev_info_data, int *api, int *sub_api)
{
	// Precedence for filter drivers vs driver is in the order of this array
	struct driver_lookup lookup[3] = {
		{"\0\0", SPDRP_SERVICE, "driver"},
		{"\0\0", SPDRP_UPPERFILTERS, "upper filter driver"},
		{"\0\0", SPDRP_LOWERFILTERS, "lower filter driver"}
	};
	DWORD size, reg_type;
	unsigned k, l;
	int i, j;

	// Check the service & filter names to know the API we should use
	for (k = 0; k < 3; k++) {
		if (pSetupDiGetDeviceRegistryPropertyA(*dev_info, dev_info_data, lookup[k].reg_prop,
			&reg_type, (PBYTE)lookup[k].list, MAX_KEY_LENGTH, &size)) {
			// Turn the REG_SZ SPDRP_SERVICE into REG_MULTI_SZ
			if (lookup[k].reg_prop == SPDRP_SERVICE)
				// our buffers are MAX_KEY_LENGTH + 1 so we can overflow if needed
				lookup[k].list[strlen(lookup[k].list) + 1] = 0;

			// MULTI_SZ is a pain to work with. Turn it into something much more manageable
			// NB: none of the driver names we check against contain LIST_SEPARATOR,
			// (currently ';'), so even if an unsuported one does, it's not an issue
			for (l = 0; (lookup[k].list[l] != 0) || (lookup[k].list[l + 1] != 0); l++) {
				if (lookup[k].list[l] == 0)
					lookup[k].list[l] = LIST_SEPARATOR;
			}
			usbi_dbg("%s(s): %s", lookup[k].designation, lookup[k].list);
		} else {
			if (GetLastError() != ERROR_INVALID_DATA)
				usbi_dbg("could not access %s: %s", lookup[k].designation, windows_error_str(0));
			lookup[k].list[0] = 0;
		}
	}

	for (i = 2; i < USB_API_MAX; i++) {
		for (k = 0; k < 3; k++) {
			j = get_sub_api(lookup[k].list, i);
			if (j >= 0) {
				usbi_dbg("matched %s name against %s", lookup[k].designation,
					(i != USB_API_WINUSBX) ? usb_api_backend[i].designation : usb_api_backend[i].driver_name_list[j]);
				*api = i;
				*sub_api = j;
				return;
			}
		}
	}
}

static int set_composite_interface(struct libusb_context *ctx, struct libusb_device *dev,
	char *dev_interface_path, char *device_id, int api, int sub_api)
{
	struct winusb_device_priv *priv = _device_priv(dev);
	int interface_number;
	const char *mi_str;

	// Because MI_## are not necessarily in sequential order (some composite
	// devices will have only MI_00 & MI_03 for instance), we retrieve the actual
	// interface number from the path's MI value
	mi_str = strstr(device_id, "MI_");
	if ((mi_str != NULL) && isdigit(mi_str[3]) && isdigit(mi_str[4])) {
		interface_number = ((mi_str[3] - '0') * 10) + (mi_str[4] - '0');
	} else {
		usbi_warn(ctx, "failure to read interface number for %s, using default value", device_id);
		interface_number = 0;
	}

	if (interface_number >= USB_MAXINTERFACES) {
		usbi_warn(ctx, "interface %d too large - ignoring interface path %s", interface_number, dev_interface_path);
		return LIBUSB_ERROR_ACCESS;
	}

	if (priv->usb_interface[interface_number].path != NULL) {
		if (api == USB_API_HID) {
			// HID devices can have multiple collections (COL##) for each MI_## interface
			usbi_dbg("interface[%d] already set - ignoring HID collection: %s",
				interface_number, device_id);
			return LIBUSB_ERROR_ACCESS;
		}
		// In other cases, just use the latest data
		safe_free(priv->usb_interface[interface_number].path);
	}

	usbi_dbg("interface[%d] = %s", interface_number, dev_interface_path);
	priv->usb_interface[interface_number].path = dev_interface_path;
	priv->usb_interface[interface_number].apib = &usb_api_backend[api];
	priv->usb_interface[interface_number].sub_api = sub_api;
	if ((api == USB_API_HID) && (priv->hid == NULL)) {
		priv->hid = calloc(1, sizeof(struct hid_device_priv));
		if (priv->hid == NULL)
			return LIBUSB_ERROR_NO_MEM;
	}

	return LIBUSB_SUCCESS;
}

static int set_hid_interface(struct libusb_context *ctx, struct libusb_device *dev,
	char *dev_interface_path)
{
	int i;
	struct winusb_device_priv *priv = _device_priv(dev);

	if (priv->hid == NULL) {
		usbi_err(ctx, "program assertion failed: parent is not HID");
		return LIBUSB_ERROR_NO_DEVICE;
	} else if (priv->hid->nb_interfaces == USB_MAXINTERFACES) {
		usbi_err(ctx, "program assertion failed: max USB interfaces reached for HID device");
		return LIBUSB_ERROR_NO_DEVICE;
	}

	for (i = 0; i < priv->hid->nb_interfaces; i++) {
		if ((priv->usb_interface[i].path != NULL) && strcmp(priv->usb_interface[i].path, dev_interface_path) == 0) {
			usbi_dbg("interface[%d] already set to %s", i, dev_interface_path);
			return LIBUSB_ERROR_ACCESS;
		}
	}

	priv->usb_interface[priv->hid->nb_interfaces].path = dev_interface_path;
	priv->usb_interface[priv->hid->nb_interfaces].apib = &usb_api_backend[USB_API_HID];
	usbi_dbg("interface[%u] = %s", priv->hid->nb_interfaces, dev_interface_path);
	priv->hid->nb_interfaces++;
	return LIBUSB_SUCCESS;
}

/*
 * get_device_list: libusb backend device enumeration function
 */
static int winusb_get_device_list(struct libusb_context *ctx, struct discovered_devs **_discdevs)
{
	struct discovered_devs *discdevs;
	HDEVINFO *dev_info, dev_info_intf, dev_info_enum;
	SP_DEVINFO_DATA dev_info_data;
	DWORD _index = 0;
	GUID hid_guid;
	int r = LIBUSB_SUCCESS;
	int api, sub_api;
	unsigned int pass, i, j;
	char enumerator[16];
	char dev_id[MAX_PATH_LENGTH];
	struct libusb_device *dev, *parent_dev;
	struct winusb_device_priv *priv, *parent_priv;
	char *dev_interface_path = NULL;
	unsigned long session_id;
	DWORD size, port_nr, reg_type, install_state;
	HKEY key;
	WCHAR guid_string_w[MAX_GUID_STRING_LENGTH];
	GUID *if_guid;
	LONG s;
#define HUB_PASS 0
#define DEV_PASS 1
#define HCD_PASS 2
#define GEN_PASS 3
#define HID_PASS 4
#define EXT_PASS 5
	// Keep a list of guids that will be enumerated
#define GUID_SIZE_STEP 8
	const GUID **guid_list, **new_guid_list;
	unsigned int guid_size = GUID_SIZE_STEP;
	unsigned int nb_guids;
	// Keep a list of PnP enumerator strings that are found
	char *usb_enumerator[8] = { "USB" };
	unsigned int nb_usb_enumerators = 1;
	unsigned int usb_enum_index = 0;
	// Keep a list of newly allocated devs to unref
#define UNREF_SIZE_STEP 16
	libusb_device **unref_list, **new_unref_list;
	unsigned int unref_size = UNREF_SIZE_STEP;
	unsigned int unref_cur = 0;

	// PASS 1 : (re)enumerate HCDs (allows for HCD hotplug)
	// PASS 2 : (re)enumerate HUBS
	// PASS 3 : (re)enumerate generic USB devices (including driverless)
	//           and list additional USB device interface GUIDs to explore
	// PASS 4 : (re)enumerate master USB devices that have a device interface
	// PASS 5+: (re)enumerate device interfaced GUIDs (including HID) and
	//           set the device interfaces.

	// Init the GUID table
	guid_list = malloc(guid_size * sizeof(void *));
	if (guid_list == NULL) {
		usbi_err(ctx, "failed to alloc guid list");
		return LIBUSB_ERROR_NO_MEM;
	}

	guid_list[HUB_PASS] = &GUID_DEVINTERFACE_USB_HUB;
	guid_list[DEV_PASS] = &GUID_DEVINTERFACE_USB_DEVICE;
	guid_list[HCD_PASS] = &GUID_DEVINTERFACE_USB_HOST_CONTROLLER;
	guid_list[GEN_PASS] = NULL;
	if (api_hid_available) {
		HidD_GetHidGuid(&hid_guid);
		guid_list[HID_PASS] = &hid_guid;
	} else {
		guid_list[HID_PASS] = NULL;
	}
	nb_guids = EXT_PASS;

	unref_list = malloc(unref_size * sizeof(void *));
	if (unref_list == NULL) {
		usbi_err(ctx, "failed to alloc unref list");
		free((void *)guid_list);
		return LIBUSB_ERROR_NO_MEM;
	}

	dev_info_intf = pSetupDiGetClassDevsA(NULL, NULL, NULL, DIGCF_ALLCLASSES | DIGCF_PRESENT | DIGCF_DEVICEINTERFACE);
	if (dev_info_intf == INVALID_HANDLE_VALUE) {
		usbi_err(ctx, "failed to obtain device info list: %s", windows_error_str(0));
		free(unref_list);
		free((void *)guid_list);
		return LIBUSB_ERROR_OTHER;
	}

	for (pass = 0; ((pass < nb_guids) && (r == LIBUSB_SUCCESS)); pass++) {
//#define ENUM_DEBUG
#if defined(ENABLE_LOGGING) && defined(ENUM_DEBUG)
		const char * const passname[] = {"HUB", "DEV", "HCD", "GEN", "HID", "EXT"};
		usbi_dbg("#### PROCESSING %ss %s", passname[MIN(pass, EXT_PASS)], guid_to_string(guid_list[pass]));
#endif
		if ((pass == HID_PASS) && (guid_list[HID_PASS] == NULL))
			continue;

		dev_info = (pass != GEN_PASS) ? &dev_info_intf : &dev_info_enum;

		for (i = 0; ; i++) {
			// safe loop: free up any (unprotected) dynamic resource
			// NB: this is always executed before breaking the loop
			safe_free(dev_interface_path);
			priv = parent_priv = NULL;
			dev = parent_dev = NULL;

			// Safe loop: end of loop conditions
			if (r != LIBUSB_SUCCESS)
				break;

			if ((pass == HCD_PASS) && (i == UINT8_MAX)) {
				usbi_warn(ctx, "program assertion failed - found more than %u buses, skipping the rest.", UINT8_MAX);
				break;
			}

			if (pass != GEN_PASS) {
				// Except for GEN, all passes deal with device interfaces
				r = get_interface_details(ctx, *dev_info, &dev_info_data, guid_list[pass], &_index, &dev_interface_path);
				if ((r != LIBUSB_SUCCESS) || (dev_interface_path == NULL)) {
					_index = 0;
					break;
				}
			} else {
				// Workaround for a Nec/Renesas USB 3.0 driver bug where root hubs are
				// being listed under the "NUSB3" PnP Symbolic Name rather than "USB".
				// The Intel USB 3.0 driver behaves similar, but uses "IUSB3"
				// The Intel Alpine Ridge USB 3.1 driver uses "IARUSB3"
				for (; usb_enum_index < nb_usb_enumerators; usb_enum_index++) {
					if (get_devinfo_data(ctx, dev_info, &dev_info_data, usb_enumerator[usb_enum_index], i))
						break;
					i = 0;
				}
				if (usb_enum_index == nb_usb_enumerators)
					break;
			}

			// Read the Device ID path
			if (!pSetupDiGetDeviceInstanceIdA(*dev_info, &dev_info_data, dev_id, sizeof(dev_id), NULL)) {
				usbi_warn(ctx, "could not read the device instance ID for devInst %X, skipping",
					  dev_info_data.DevInst);
				continue;
			}

#ifdef ENUM_DEBUG
			usbi_dbg("PRO: %s", dev_id);
#endif

			// Set API to use or get additional data from generic pass
			api = USB_API_UNSUPPORTED;
			sub_api = SUB_API_NOTSET;
			switch (pass) {
			case HCD_PASS:
				break;
			case HUB_PASS:
				api = USB_API_HUB;
				// Fetch the PnP enumerator class for this hub
				// This will allow us to enumerate all classes during the GEN pass
				if (!pSetupDiGetDeviceRegistryPropertyA(*dev_info, &dev_info_data, SPDRP_ENUMERATOR_NAME,
					NULL, (PBYTE)enumerator, sizeof(enumerator), NULL)) {
					usbi_err(ctx, "could not read enumerator string for device '%s': %s", dev_id, windows_error_str(0));
					LOOP_BREAK(LIBUSB_ERROR_OTHER);
				}
				for (j = 0; j < nb_usb_enumerators; j++) {
					if (strcmp(usb_enumerator[j], enumerator) == 0)
						break;
				}
				if (j == nb_usb_enumerators) {
					usbi_dbg("found new PnP enumerator string '%s'", enumerator);
					if (nb_usb_enumerators < ARRAYSIZE(usb_enumerator)) {
						usb_enumerator[nb_usb_enumerators] = _strdup(enumerator);
						if (usb_enumerator[nb_usb_enumerators] != NULL) {
							nb_usb_enumerators++;
						} else {
							usbi_err(ctx, "could not allocate enumerator string '%s'", enumerator);
							LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
						}
					} else {
						usbi_warn(ctx, "too many enumerator strings, some devices may not be accessible");
					}
				}
				break;
			case GEN_PASS:
				// We use the GEN pass to detect driverless devices...
				if (!pSetupDiGetDeviceRegistryPropertyA(*dev_info, &dev_info_data, SPDRP_DRIVER,
					NULL, NULL, 0, NULL) && (GetLastError() != ERROR_INSUFFICIENT_BUFFER)) {
					usbi_info(ctx, "The following device has no driver: '%s'", dev_id);
					usbi_info(ctx, "libusb will not be able to access it");
				}
				// ...and to add the additional device interface GUIDs
				key = pSetupDiOpenDevRegKey(*dev_info, &dev_info_data, DICS_FLAG_GLOBAL, 0, DIREG_DEV, KEY_READ);
				if (key == INVALID_HANDLE_VALUE)
					break;
				// Look for both DeviceInterfaceGUIDs *and* DeviceInterfaceGUID, in that order
				size = sizeof(guid_string_w);
				s = pRegQueryValueExW(key, L"DeviceInterfaceGUIDs", NULL, &reg_type,
					(LPBYTE)guid_string_w, &size);
				if (s == ERROR_FILE_NOT_FOUND)
					s = pRegQueryValueExW(key, L"DeviceInterfaceGUID", NULL, &reg_type,
						(LPBYTE)guid_string_w, &size);
				pRegCloseKey(key);
				if ((s == ERROR_SUCCESS) &&
				    (((reg_type == REG_SZ) && (size == (sizeof(guid_string_w) - sizeof(WCHAR)))) ||
				     ((reg_type == REG_MULTI_SZ) && (size == sizeof(guid_string_w))))) {
					if (nb_guids == guid_size) {
						new_guid_list = realloc((void *)guid_list, (guid_size + GUID_SIZE_STEP) * sizeof(void *));
						if (new_guid_list == NULL) {
							usbi_err(ctx, "failed to realloc guid list");
							LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
						}
						guid_list = new_guid_list;
						guid_size += GUID_SIZE_STEP;
					}
					if_guid = malloc(sizeof(*if_guid));
					if (if_guid == NULL) {
						usbi_err(ctx, "failed to alloc if_guid");
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
					}
					if (pIIDFromString(guid_string_w, if_guid) != 0) {
						usbi_warn(ctx, "device '%s' has malformed DeviceInterfaceGUID string, skipping", dev_id);
						free(if_guid);
					} else {
						// Check if we've already seen this GUID
						for (j = EXT_PASS; j < nb_guids; j++) {
							if (memcmp(guid_list[j], if_guid, sizeof(*if_guid)) == 0)
								break;
						}
						if (j == nb_guids) {
							usbi_dbg("extra GUID: %s", guid_to_string(if_guid));
							guid_list[nb_guids++] = if_guid;
						} else {
							// Duplicate, ignore
							free(if_guid);
						}
					}
				} else if (s == ERROR_SUCCESS) {
					usbi_warn(ctx, "unexpected type/size of DeviceInterfaceGUID for '%s'", dev_id);
				}
				break;
			case HID_PASS:
				api = USB_API_HID;
				break;
			default:
				// Get the API type (after checking that the driver installation is OK)
				if ((!pSetupDiGetDeviceRegistryPropertyA(*dev_info, &dev_info_data, SPDRP_INSTALL_STATE,
					NULL, (PBYTE)&install_state, sizeof(install_state), &size)) || (size != sizeof(install_state))) {
					usbi_warn(ctx, "could not detect installation state of driver for '%s': %s",
						dev_id, windows_error_str(0));
				} else if (install_state != 0) {
					usbi_warn(ctx, "driver for device '%s' is reporting an issue (code: %u) - skipping",
						dev_id, (unsigned int)install_state);
					continue;
				}
				get_api_type(ctx, dev_info, &dev_info_data, &api, &sub_api);
				break;
			}

			// Find parent device (for the passes that need it)
			if (pass >= GEN_PASS) {
				parent_dev = get_ancestor(ctx, dev_info_data.DevInst, NULL);
				if (parent_dev == NULL) {
					// Root hubs will not have a parent
					dev = usbi_get_device_by_session_id(ctx, (unsigned long)dev_info_data.DevInst);
					if (dev != NULL) {
						priv = _device_priv(dev);
						if (priv->root_hub)
							goto track_unref;
						libusb_unref_device(dev);
					}

					usbi_dbg("unlisted ancestor for '%s' (non USB HID, newly connected, etc.) - ignoring", dev_id);
					continue;
				}

				parent_priv = _device_priv(parent_dev);
				// virtual USB devices are also listed during GEN - don't process these yet
				if ((pass == GEN_PASS) && (parent_priv->apib->id != USB_API_HUB)) {
					libusb_unref_device(parent_dev);
					continue;
				}
			}

			// Create new or match existing device, using the devInst as session id
			if ((pass <= GEN_PASS) && (pass != HCD_PASS)) {	// For subsequent passes, we'll lookup the parent
				// These are the passes that create "new" devices
				session_id = (unsigned long)dev_info_data.DevInst;
				dev = usbi_get_device_by_session_id(ctx, session_id);
				if (dev == NULL) {
				alloc_device:
					usbi_dbg("allocating new device for session [%lX]", session_id);
					dev = usbi_alloc_device(ctx, session_id);
					if (dev == NULL)
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);

					priv = winusb_device_priv_init(dev);
					priv->dev_id = _strdup(dev_id);
					if (priv->dev_id == NULL) {
						libusb_unref_device(dev);
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
					}
				} else {
					usbi_dbg("found existing device for session [%lX]", session_id);

					priv = _device_priv(dev);
					if (strcmp(priv->dev_id, dev_id) != 0) {
						usbi_dbg("device instance ID for session [%lX] changed", session_id);
						usbi_disconnect_device(dev);
						libusb_unref_device(dev);
						goto alloc_device;
					}
				}

			track_unref:
				// Keep track of devices that need unref
				if (unref_cur == unref_size) {
					new_unref_list = realloc(unref_list, (unref_size + UNREF_SIZE_STEP) * sizeof(void *));
					if (new_unref_list == NULL) {
						usbi_err(ctx, "could not realloc list for unref - aborting");
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
					}
					unref_list = new_unref_list;
					unref_size += UNREF_SIZE_STEP;
				}
				unref_list[unref_cur++] = dev;
			}

			// Setup device
			switch (pass) {
			case HUB_PASS:
			case DEV_PASS:
				// If the device has already been setup, don't do it again
				if (priv->path != NULL)
					break;
				// Take care of API initialization
				priv->path = dev_interface_path;
				dev_interface_path = NULL;
				priv->apib = &usb_api_backend[api];
				priv->sub_api = sub_api;
				switch (api) {
				case USB_API_COMPOSITE:
				case USB_API_HUB:
					break;
				case USB_API_HID:
					priv->hid = calloc(1, sizeof(struct hid_device_priv));
					if (priv->hid == NULL)
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
					break;
				default:
					// For other devices, the first interface is the same as the device
					priv->usb_interface[0].path = _strdup(priv->path);
					if (priv->usb_interface[0].path == NULL)
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);
					// The following is needed if we want API calls to work for both simple
					// and composite devices.
					for (j = 0; j < USB_MAXINTERFACES; j++)
						priv->usb_interface[j].apib = &usb_api_backend[api];
					break;
				}
				break;
			case HCD_PASS:
				r = enumerate_hcd_root_hub(ctx, dev_id, (uint8_t)(i + 1), dev_info_data.DevInst);
				break;
			case GEN_PASS:
				// The SPDRP_ADDRESS for USB devices is the device port number on the hub
				port_nr = 0;
				if (!pSetupDiGetDeviceRegistryPropertyA(*dev_info, &dev_info_data, SPDRP_ADDRESS,
						NULL, (PBYTE)&port_nr, sizeof(port_nr), &size) || (size != sizeof(port_nr)))
					usbi_warn(ctx, "could not retrieve port number for device '%s': %s", dev_id, windows_error_str(0));
				r = init_device(dev, parent_dev, (uint8_t)port_nr, dev_info_data.DevInst);
				if (r == LIBUSB_SUCCESS) {
					// Append device to the list of discovered devices
					discdevs = discovered_devs_append(*_discdevs, dev);
					if (!discdevs)
						LOOP_BREAK(LIBUSB_ERROR_NO_MEM);

					*_discdevs = discdevs;
				} else if (r == LIBUSB_ERROR_NO_DEVICE) {
					// This can occur if the device was disconnected but Windows hasn't
					// refreshed its enumeration yet - in that case, we ignore the device
					r = LIBUSB_SUCCESS;
				}
				break;
			default: // HID_PASS and later
				if (parent_priv->apib->id == USB_API_HID || parent_priv->apib->id == USB_API_COMPOSITE) {
					if (parent_priv->apib->id == USB_API_HID) {
						usbi_dbg("setting HID interface for [%lX]:", parent_dev->session_data);
						r = set_hid_interface(ctx, parent_dev, dev_interface_path);
					} else {
						usbi_dbg("setting composite interface for [%lX]:", parent_dev->session_data);
						r = set_composite_interface(ctx, parent_dev, dev_interface_path, dev_id, api, sub_api);
					}
					switch (r) {
					case LIBUSB_SUCCESS:
						dev_interface_path = NULL;
						break;
					case LIBUSB_ERROR_ACCESS:
						// interface has already been set => make sure dev_interface_path is freed then
						r = LIBUSB_SUCCESS;
						break;
					default:
						LOOP_BREAK(r);
						break;
					}
				}
				libusb_unref_device(parent_dev);
				break;
			}
		}
	}

	pSetupDiDestroyDeviceInfoList(dev_info_intf);

	// Free any additional GUIDs
	for (pass = EXT_PASS; pass < nb_guids; pass++)
		free((void *)guid_list[pass]);
	free((void *)guid_list);

	// Free any PnP enumerator strings
	for (i = 1; i < nb_usb_enumerators; i++)
		free(usb_enumerator[i]);

	// Unref newly allocated devs
	for (i = 0; i < unref_cur; i++)
		libusb_unref_device(unref_list[i]);
	free(unref_list);

	return r;
}

static int winusb_get_device_descriptor(struct libusb_device *dev, unsigned char *buffer)
{
	struct winusb_device_priv *priv = _device_priv(dev);

	memcpy(buffer, &priv->dev_descriptor, DEVICE_DESC_LENGTH);
	return LIBUSB_SUCCESS;
}

static int winusb_get_config_descriptor(struct libusb_device *dev, uint8_t config_index, unsigned char *buffer, size_t len)
{
	struct winusb_device_priv *priv = _device_priv(dev);
	PUSB_CONFIGURATION_DESCRIPTOR config_header;
	size_t size;

	// config index is zero based
	if (config_index >= dev->num_configurations)
		return LIBUSB_ERROR_INVALID_PARAM;

	if ((priv->config_descriptor == NULL) || (priv->config_descriptor[config_index] == NULL))
		return LIBUSB_ERROR_NOT_FOUND;

	config_header = priv->config_descriptor[config_index];

	size = MIN(config_header->wTotalLength, len);
	memcpy(buffer, priv->config_descriptor[config_index], size);
	return (int)size;
}

static int winusb_get_config_descriptor_by_value(struct libusb_device *dev, uint8_t bConfigurationValue,
	unsigned char **buffer)
{
	struct winusb_device_priv *priv = _device_priv(dev);
	PUSB_CONFIGURATION_DESCRIPTOR config_header;
	uint8_t index;

	if (priv->config_descriptor == NULL)
		return LIBUSB_ERROR_NOT_FOUND;

	for (index = 0; index < dev->num_configurations; index++) {
		config_header = priv->config_descriptor[index];
		if (config_header == NULL)
			continue;
		if (config_header->bConfigurationValue == bConfigurationValue) {
			*buffer = (unsigned char *)priv->config_descriptor[index];
			return (int)config_header->wTotalLength;
		}
	}

	return LIBUSB_ERROR_NOT_FOUND;
}

/*
 * return the cached copy of the active config descriptor
 */
static int winusb_get_active_config_descriptor(struct libusb_device *dev, unsigned char *buffer, size_t len)
{
	struct winusb_device_priv *priv = _device_priv(dev);
	unsigned char *config_desc;
	int r;

	if (priv->active_config == 0)
		return LIBUSB_ERROR_NOT_FOUND;

	r = winusb_get_config_descriptor_by_value(dev, priv->active_config, &config_desc);
	if (r < 0)
		return r;

	len = MIN((size_t)r, len);
	memcpy(buffer, config_desc, len);
	return (int)len;
}

static int winusb_open(struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, open);

	return priv->apib->open(SUB_API_NOTSET, dev_handle);
}

static void winusb_close(struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	if (priv->apib->close)
		priv->apib->close(SUB_API_NOTSET, dev_handle);
}

static int winusb_get_configuration(struct libusb_device_handle *dev_handle, int *config)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	if (priv->active_config == 0) {
		*config = 0;
		return LIBUSB_ERROR_NOT_FOUND;
	}

	*config = priv->active_config;
	return LIBUSB_SUCCESS;
}

/*
 * from http://msdn.microsoft.com/en-us/library/ms793522.aspx: "The port driver
 * does not currently expose a service that allows higher-level drivers to set
 * the configuration."
 */
static int winusb_set_configuration(struct libusb_device_handle *dev_handle, int config)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int r = LIBUSB_SUCCESS;

	if (config >= USB_MAXCONFIG)
		return LIBUSB_ERROR_INVALID_PARAM;

	r = libusb_control_transfer(dev_handle, LIBUSB_ENDPOINT_OUT |
		LIBUSB_REQUEST_TYPE_STANDARD | LIBUSB_RECIPIENT_DEVICE,
		LIBUSB_REQUEST_SET_CONFIGURATION, (uint16_t)config,
		0, NULL, 0, 1000);

	if (r == LIBUSB_SUCCESS)
		priv->active_config = (uint8_t)config;

	return r;
}

static int winusb_claim_interface(struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int r;

	CHECK_SUPPORTED_API(priv->apib, claim_interface);

	safe_free(priv->usb_interface[iface].endpoint);
	priv->usb_interface[iface].nb_endpoints = 0;

	r = priv->apib->claim_interface(SUB_API_NOTSET, dev_handle, iface);

	if (r == LIBUSB_SUCCESS)
		r = windows_assign_endpoints(dev_handle, iface, 0);

	return r;
}

static int winusb_set_interface_altsetting(struct libusb_device_handle *dev_handle, int iface, int altsetting)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int r;

	CHECK_SUPPORTED_API(priv->apib, set_interface_altsetting);

	safe_free(priv->usb_interface[iface].endpoint);
	priv->usb_interface[iface].nb_endpoints = 0;

	r = priv->apib->set_interface_altsetting(SUB_API_NOTSET, dev_handle, iface, altsetting);

	if (r == LIBUSB_SUCCESS)
		r = windows_assign_endpoints(dev_handle, iface, altsetting);

	return r;
}

static int winusb_release_interface(struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, release_interface);

	return priv->apib->release_interface(SUB_API_NOTSET, dev_handle, iface);
}

static int winusb_clear_halt(struct libusb_device_handle *dev_handle, unsigned char endpoint)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, clear_halt);

	return priv->apib->clear_halt(SUB_API_NOTSET, dev_handle, endpoint);
}

static int winusb_reset_device(struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, reset_device);

	return priv->apib->reset_device(SUB_API_NOTSET, dev_handle);
}

static void winusb_destroy_device(struct libusb_device *dev)
{
	winusb_device_priv_release(dev);
}

static void winusb_clear_transfer_priv(struct usbi_transfer *itransfer)
{
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);

	usbi_close(transfer_priv->pollable_fd.fd);
	transfer_priv->pollable_fd = INVALID_WINFD;
	transfer_priv->handle = NULL;
	safe_free(transfer_priv->hid_buffer);
	safe_free(transfer_priv->iso_context);

	// When auto claim is in use, attempt to release the auto-claimed interface
	auto_release(itransfer);
}

static int do_submit_transfer(struct usbi_transfer *itransfer, short events,
	int (*transfer_fn)(int, struct usbi_transfer *))
{
	struct libusb_context *ctx = ITRANSFER_CTX(itransfer);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winfd wfd;
	int r;

	wfd = usbi_create_fd();
	if (wfd.fd < 0)
		return LIBUSB_ERROR_NO_MEM;

	r = usbi_add_pollfd(ctx, wfd.fd, events);
	if (r) {
		usbi_close(wfd.fd);
		return r;
	}

	// Use transfer_priv to store data needed for async polling
	transfer_priv->pollable_fd = wfd;

	r = transfer_fn(SUB_API_NOTSET, itransfer);

	if ((r != LIBUSB_SUCCESS) && (r != LIBUSB_ERROR_OVERFLOW)) {
		usbi_remove_pollfd(ctx, wfd.fd);
		usbi_close(wfd.fd);
		transfer_priv->pollable_fd = INVALID_WINFD;
	}

	return r;
}

static int winusb_submit_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int (*transfer_fn)(int, struct usbi_transfer *);
	short events;

	switch (transfer->type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
		events = (transfer->buffer[0] & LIBUSB_ENDPOINT_IN) ? POLLIN : POLLOUT;
		transfer_fn = priv->apib->submit_control_transfer;
		break;
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
		if (IS_XFEROUT(transfer) && (transfer->flags & LIBUSB_TRANSFER_ADD_ZERO_PACKET))
			return LIBUSB_ERROR_NOT_SUPPORTED;
		events = IS_XFERIN(transfer) ? POLLIN : POLLOUT;
		transfer_fn = priv->apib->submit_bulk_transfer;
		break;
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		events = IS_XFERIN(transfer) ? POLLIN : POLLOUT;
		transfer_fn = priv->apib->submit_iso_transfer;
		break;
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		return LIBUSB_ERROR_NOT_SUPPORTED;
	default:
		usbi_err(TRANSFER_CTX(transfer), "unknown endpoint type %d", transfer->type);
		return LIBUSB_ERROR_INVALID_PARAM;
	}

	if (transfer_fn == NULL) {
		usbi_warn(TRANSFER_CTX(transfer),
			"unsupported transfer type %d (unrecognized device driver)",
			transfer->type);
		return LIBUSB_ERROR_NOT_SUPPORTED;
	}

	return do_submit_transfer(itransfer, events, transfer_fn);
}

static int windows_abort_control(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, abort_control);

	return priv->apib->abort_control(SUB_API_NOTSET, itransfer);
}

static int windows_abort_transfers(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);

	CHECK_SUPPORTED_API(priv->apib, abort_transfers);

	return priv->apib->abort_transfers(SUB_API_NOTSET, itransfer);
}

static int winusb_cancel_transfer(struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);

	switch (transfer->type) {
	case LIBUSB_TRANSFER_TYPE_CONTROL:
		return windows_abort_control(itransfer);
	case LIBUSB_TRANSFER_TYPE_BULK:
	case LIBUSB_TRANSFER_TYPE_INTERRUPT:
	case LIBUSB_TRANSFER_TYPE_ISOCHRONOUS:
		return windows_abort_transfers(itransfer);
	case LIBUSB_TRANSFER_TYPE_BULK_STREAM:
		return LIBUSB_ERROR_NOT_SUPPORTED;
	default:
		usbi_err(ITRANSFER_CTX(itransfer), "unknown endpoint type %d", transfer->type);
		return LIBUSB_ERROR_INVALID_PARAM;
	}
}

static int winusb_copy_transfer_data(struct usbi_transfer *itransfer, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	return priv->apib->copy_transfer_data(SUB_API_NOTSET, itransfer, io_size);
}

static int winusb_get_transfer_fd(struct usbi_transfer *itransfer)
{
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	return transfer_priv->pollable_fd.fd;
}

static void winusb_get_overlapped_result(struct usbi_transfer *itransfer,
	DWORD *io_result, DWORD *io_size)
{
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winfd *pollable_fd = &transfer_priv->pollable_fd;

	if (HasOverlappedIoCompletedSync(pollable_fd->overlapped)) {
		*io_result = NO_ERROR;
		*io_size = (DWORD)pollable_fd->overlapped->InternalHigh;
	} else if (GetOverlappedResult(transfer_priv->handle, pollable_fd->overlapped, io_size, FALSE)) {
		// Regular async overlapped
		*io_result = NO_ERROR;
	} else {
		*io_result = GetLastError();
	}
}

// NB: MSVC6 does not support named initializers.
const struct windows_backend winusb_backend = {
	winusb_init,
	winusb_exit,
	winusb_get_device_list,
	winusb_open,
	winusb_close,
	winusb_get_device_descriptor,
	winusb_get_active_config_descriptor,
	winusb_get_config_descriptor,
	winusb_get_config_descriptor_by_value,
	winusb_get_configuration,
	winusb_set_configuration,
	winusb_claim_interface,
	winusb_release_interface,
	winusb_set_interface_altsetting,
	winusb_clear_halt,
	winusb_reset_device,
	winusb_destroy_device,
	winusb_submit_transfer,
	winusb_cancel_transfer,
	winusb_clear_transfer_priv,
	winusb_copy_transfer_data,
	winusb_get_transfer_fd,
	winusb_get_overlapped_result,
};

/*
 * USB API backends
 */

static const char * const composite_driver_names[] = {"USBCCGP"};
static const char * const winusbx_driver_names[] = {"libusbK", "libusb0", "WinUSB"};
static const char * const hid_driver_names[] = {"HIDUSB", "MOUHID", "KBDHID"};
const struct windows_usb_api_backend usb_api_backend[USB_API_MAX] = {
	{
		USB_API_UNSUPPORTED,
		"Unsupported API",
		// No supported operations
	},
	{
		USB_API_HUB,
		"HUB API",
		// No supported operations
	},
	{
		USB_API_COMPOSITE,
		"Composite API",
		composite_driver_names,
		ARRAYSIZE(composite_driver_names),
		NULL,	/* init */
		NULL,	/* exit */
		composite_open,
		composite_close,
		NULL,	/* configure_endpoints */
		composite_claim_interface,
		composite_set_interface_altsetting,
		composite_release_interface,
		composite_clear_halt,
		composite_reset_device,
		composite_submit_bulk_transfer,
		composite_submit_iso_transfer,
		composite_submit_control_transfer,
		composite_abort_control,
		composite_abort_transfers,
		composite_copy_transfer_data,
	},
	{
		USB_API_WINUSBX,
		"WinUSB-like APIs",
		winusbx_driver_names,
		ARRAYSIZE(winusbx_driver_names),
		winusbx_init,
		winusbx_exit,
		winusbx_open,
		winusbx_close,
		winusbx_configure_endpoints,
		winusbx_claim_interface,
		winusbx_set_interface_altsetting,
		winusbx_release_interface,
		winusbx_clear_halt,
		winusbx_reset_device,
		winusbx_submit_bulk_transfer,
		winusbx_submit_iso_transfer,
		winusbx_submit_control_transfer,
		winusbx_abort_control,
		winusbx_abort_transfers,
		winusbx_copy_transfer_data,
	},
	{
		USB_API_HID,
		"HID API",
		// No supported operations
	},
};


/*
 * WinUSB-like (WinUSB, libusb0/libusbK through libusbk DLL) API functions
 */
#define WinUSBX_Set(fn)										\
	do {											\
		if (native_winusb)								\
			WinUSBX[i].fn = (WinUsb_##fn##_t)GetProcAddress(h, "WinUsb_" #fn);	\
		else										\
			pLibK_GetProcAddress((PVOID *)&WinUSBX[i].fn, i, KUSB_FNID_##fn);	\
	} while (0)

static int winusbx_init(struct libusb_context *ctx)
{
	HMODULE h;
	bool native_winusb;
	int i;
	KLIB_VERSION LibK_Version;
	LibK_GetProcAddress_t pLibK_GetProcAddress = NULL;
	LibK_GetVersion_t pLibK_GetVersion;

	h = LoadLibraryA("libusbK");

	if (h == NULL) {
		usbi_info(ctx, "libusbK DLL is not available, will use native WinUSB");
		h = LoadLibraryA("WinUSB");

		if (h == NULL) {
			usbi_warn(ctx, "WinUSB DLL is not available either, "
				"you will not be able to access devices outside of enumeration");
			return LIBUSB_ERROR_NOT_FOUND;
		}
	} else {
		usbi_dbg("using libusbK DLL for universal access");
		pLibK_GetVersion = (LibK_GetVersion_t)GetProcAddress(h, "LibK_GetVersion");
		if (pLibK_GetVersion != NULL) {
			pLibK_GetVersion(&LibK_Version);
			usbi_dbg("libusbK version: %d.%d.%d.%d", LibK_Version.Major, LibK_Version.Minor,
				LibK_Version.Micro, LibK_Version.Nano);
		}
		pLibK_GetProcAddress = (LibK_GetProcAddress_t)GetProcAddress(h, "LibK_GetProcAddress");
		if (pLibK_GetProcAddress == NULL) {
			usbi_err(ctx, "LibK_GetProcAddress() not found in libusbK DLL");
			FreeLibrary(h);
			return LIBUSB_ERROR_NOT_FOUND;
		}
	}

	native_winusb = (pLibK_GetProcAddress == NULL);
	for (i = 0; i < SUB_API_MAX; i++) {
		WinUSBX_Set(AbortPipe);
		WinUSBX_Set(ControlTransfer);
		WinUSBX_Set(FlushPipe);
		WinUSBX_Set(Free);
		WinUSBX_Set(GetAssociatedInterface);
		WinUSBX_Set(Initialize);
		WinUSBX_Set(ReadPipe);
		if (!native_winusb)
			WinUSBX_Set(ResetDevice);
		WinUSBX_Set(ResetPipe);
		WinUSBX_Set(SetCurrentAlternateSetting);
		WinUSBX_Set(SetPipePolicy);
		WinUSBX_Set(WritePipe);
		WinUSBX_Set(IsoReadPipe);
		WinUSBX_Set(IsoWritePipe);

		if (WinUSBX[i].Initialize != NULL) {
			WinUSBX[i].initialized = true;
			// Assume driver supports CancelIoEx() if it is available
			WinUSBX[i].CancelIoEx_supported = (pCancelIoEx != NULL);
			usbi_dbg("initalized sub API %s", winusbx_driver_names[i]);
		} else {
			usbi_warn(ctx, "Failed to initalize sub API %s", winusbx_driver_names[i]);
			WinUSBX[i].initialized = false;
		}
	}

	WinUSBX_handle = h;
	return LIBUSB_SUCCESS;
}

static void winusbx_exit(void)
{
	if (WinUSBX_handle != NULL) {
		FreeLibrary(WinUSBX_handle);
		WinUSBX_handle = NULL;

		/* Reset the WinUSBX API structures */
		memset(&WinUSBX, 0, sizeof(WinUSBX));
	}
}

// NB: open and close must ensure that they only handle interface of
// the right API type, as these functions can be called wholesale from
// composite_open(), with interfaces belonging to different APIs
static int winusbx_open(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	HANDLE file_handle;
	int i;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	// WinUSB requires a separate handle for each interface
	for (i = 0; i < USB_MAXINTERFACES; i++) {
		if ((priv->usb_interface[i].path != NULL)
				&& (priv->usb_interface[i].apib->id == USB_API_WINUSBX)) {
			file_handle = CreateFileA(priv->usb_interface[i].path, GENERIC_WRITE | GENERIC_READ, FILE_SHARE_WRITE | FILE_SHARE_READ,
				NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL | FILE_FLAG_OVERLAPPED, NULL);
			if (file_handle == INVALID_HANDLE_VALUE) {
				usbi_err(ctx, "could not open device %s (interface %d): %s", priv->usb_interface[i].path, i, windows_error_str(0));
				switch (GetLastError()) {
				case ERROR_FILE_NOT_FOUND: // The device was disconnected
					return LIBUSB_ERROR_NO_DEVICE;
				case ERROR_ACCESS_DENIED:
					return LIBUSB_ERROR_ACCESS;
				default:
					return LIBUSB_ERROR_IO;
				}
			}
			handle_priv->interface_handle[i].dev_handle = file_handle;
		}
	}
	return LIBUSB_SUCCESS;
}

static void winusbx_close(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE handle;
	int i;

	if (sub_api == SUB_API_NOTSET)
		sub_api = priv->sub_api;

	if (!WinUSBX[sub_api].initialized)
		return;

	if (priv->apib->id == USB_API_COMPOSITE) {
		// If this is a composite device, just free and close all WinUSB-like
		// interfaces directly (each is independent and not associated with another)
		for (i = 0; i < USB_MAXINTERFACES; i++) {
			if (priv->usb_interface[i].apib->id == USB_API_WINUSBX) {
				handle = handle_priv->interface_handle[i].api_handle;
				if (HANDLE_VALID(handle))
					WinUSBX[sub_api].Free(handle);

				handle = handle_priv->interface_handle[i].dev_handle;
				if (HANDLE_VALID(handle))
					CloseHandle(handle);
			}
		}
	} else {
		// If this is a WinUSB device, free all interfaces above interface 0,
		// then free and close interface 0 last
		for (i = 1; i < USB_MAXINTERFACES; i++) {
			handle = handle_priv->interface_handle[i].api_handle;
			if (HANDLE_VALID(handle))
				WinUSBX[sub_api].Free(handle);
		}
		handle = handle_priv->interface_handle[0].api_handle;
		if (HANDLE_VALID(handle))
			WinUSBX[sub_api].Free(handle);

		handle = handle_priv->interface_handle[0].dev_handle;
		if (HANDLE_VALID(handle))
			CloseHandle(handle);
	}
}

static int winusbx_configure_endpoints(int sub_api, struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE winusb_handle = handle_priv->interface_handle[iface].api_handle;
	UCHAR policy;
	ULONG timeout = 0;
	uint8_t endpoint_address;
	int i;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	// With handle and enpoints set (in parent), we can setup the default pipe properties
	// see http://download.microsoft.com/download/D/1/D/D1DD7745-426B-4CC3-A269-ABBBE427C0EF/DVC-T705_DDC08.pptx
	for (i = -1; i < priv->usb_interface[iface].nb_endpoints; i++) {
		endpoint_address = (i == -1) ? 0 : priv->usb_interface[iface].endpoint[i];
		if (!WinUSBX[sub_api].SetPipePolicy(winusb_handle, endpoint_address,
			PIPE_TRANSFER_TIMEOUT, sizeof(ULONG), &timeout))
			usbi_dbg("failed to set PIPE_TRANSFER_TIMEOUT for control endpoint %02X", endpoint_address);

		if ((i == -1) || (sub_api == SUB_API_LIBUSB0))
			continue; // Other policies don't apply to control endpoint or libusb0

		policy = false;
		if (!WinUSBX[sub_api].SetPipePolicy(winusb_handle, endpoint_address,
			SHORT_PACKET_TERMINATE, sizeof(UCHAR), &policy))
			usbi_dbg("failed to disable SHORT_PACKET_TERMINATE for endpoint %02X", endpoint_address);

		if (!WinUSBX[sub_api].SetPipePolicy(winusb_handle, endpoint_address,
			IGNORE_SHORT_PACKETS, sizeof(UCHAR), &policy))
			usbi_dbg("failed to disable IGNORE_SHORT_PACKETS for endpoint %02X", endpoint_address);

		policy = true;
		/* ALLOW_PARTIAL_READS must be enabled due to likely libusbK bug. See:
		   https://sourceforge.net/mailarchive/message.php?msg_id=29736015 */
		if (!WinUSBX[sub_api].SetPipePolicy(winusb_handle, endpoint_address,
			ALLOW_PARTIAL_READS, sizeof(UCHAR), &policy))
			usbi_dbg("failed to enable ALLOW_PARTIAL_READS for endpoint %02X", endpoint_address);

		if (!WinUSBX[sub_api].SetPipePolicy(winusb_handle, endpoint_address,
			AUTO_CLEAR_STALL, sizeof(UCHAR), &policy))
			usbi_dbg("failed to enable AUTO_CLEAR_STALL for endpoint %02X", endpoint_address);
	}

	return LIBUSB_SUCCESS;
}

static int winusbx_claim_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	bool is_using_usbccgp = (priv->apib->id == USB_API_COMPOSITE);
	SP_DEVICE_INTERFACE_DETAIL_DATA_A *dev_interface_details = NULL;
	HDEVINFO dev_info = INVALID_HANDLE_VALUE;
	SP_DEVINFO_DATA dev_info_data;
	char *dev_path_no_guid = NULL;
	char filter_path[] = "\\\\.\\libusb0-0000";
	bool found_filter = false;
	HANDLE file_handle, winusb_handle;
	DWORD err;
	int i;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	// If the device is composite, but using the default Windows composite parent driver (usbccgp)
	// or if it's the first WinUSB-like interface, we get a handle through Initialize().
	if ((is_using_usbccgp) || (iface == 0)) {
		// composite device (independent interfaces) or interface 0
		file_handle = handle_priv->interface_handle[iface].dev_handle;
		if (!HANDLE_VALID(file_handle))
			return LIBUSB_ERROR_NOT_FOUND;

		if (!WinUSBX[sub_api].Initialize(file_handle, &winusb_handle)) {
			handle_priv->interface_handle[iface].api_handle = INVALID_HANDLE_VALUE;
			err = GetLastError();
			switch (err) {
			case ERROR_BAD_COMMAND:
				// The device was disconnected
				usbi_err(ctx, "could not access interface %d: %s", iface, windows_error_str(0));
				return LIBUSB_ERROR_NO_DEVICE;
			default:
				// it may be that we're using the libusb0 filter driver.
				// TODO: can we move this whole business into the K/0 DLL?
				for (i = 0; ; i++) {
					safe_free(dev_interface_details);
					safe_free(dev_path_no_guid);

					dev_interface_details = get_interface_details_filter(ctx, &dev_info, &dev_info_data, &GUID_DEVINTERFACE_LIBUSB0_FILTER, i, filter_path);
					if ((found_filter) || (dev_interface_details == NULL))
						break;

					// ignore GUID part
					dev_path_no_guid = sanitize_path(strtok(dev_interface_details->DevicePath, "{"));
					if (dev_path_no_guid == NULL)
						continue;

					if (strncmp(dev_path_no_guid, priv->usb_interface[iface].path, strlen(dev_path_no_guid)) == 0) {
						file_handle = CreateFileA(filter_path, GENERIC_WRITE | GENERIC_READ, FILE_SHARE_WRITE | FILE_SHARE_READ,
							NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL | FILE_FLAG_OVERLAPPED, NULL);
						if (file_handle != INVALID_HANDLE_VALUE) {
							if (WinUSBX[sub_api].Initialize(file_handle, &winusb_handle)) {
								// Replace the existing file handle with the working one
								CloseHandle(handle_priv->interface_handle[iface].dev_handle);
								handle_priv->interface_handle[iface].dev_handle = file_handle;
								found_filter = true;
							} else {
								usbi_err(ctx, "could not initialize filter driver for %s", filter_path);
								CloseHandle(file_handle);
							}
						} else {
							usbi_err(ctx, "could not open device %s: %s", filter_path, windows_error_str(0));
						}
					}
				}
				free(dev_interface_details);
				if (!found_filter) {
					usbi_err(ctx, "could not access interface %d: %s", iface, windows_error_str(err));
					return LIBUSB_ERROR_ACCESS;
				}
			}
		}
		handle_priv->interface_handle[iface].api_handle = winusb_handle;
	} else {
		// For all other interfaces, use GetAssociatedInterface()
		winusb_handle = handle_priv->interface_handle[0].api_handle;
		// It is a requirement for multiple interface devices on Windows that, to you
		// must first claim the first interface before you claim the others
		if (!HANDLE_VALID(winusb_handle)) {
			file_handle = handle_priv->interface_handle[0].dev_handle;
			if (WinUSBX[sub_api].Initialize(file_handle, &winusb_handle)) {
				handle_priv->interface_handle[0].api_handle = winusb_handle;
				usbi_warn(ctx, "auto-claimed interface 0 (required to claim %d with WinUSB)", iface);
			} else {
				usbi_warn(ctx, "failed to auto-claim interface 0 (required to claim %d with WinUSB): %s", iface, windows_error_str(0));
				return LIBUSB_ERROR_ACCESS;
			}
		}
		if (!WinUSBX[sub_api].GetAssociatedInterface(winusb_handle, (UCHAR)(iface - 1),
			&handle_priv->interface_handle[iface].api_handle)) {
			handle_priv->interface_handle[iface].api_handle = INVALID_HANDLE_VALUE;
			switch (GetLastError()) {
			case ERROR_NO_MORE_ITEMS:   // invalid iface
				return LIBUSB_ERROR_NOT_FOUND;
			case ERROR_BAD_COMMAND:     // The device was disconnected
				return LIBUSB_ERROR_NO_DEVICE;
			case ERROR_ALREADY_EXISTS:  // already claimed
				return LIBUSB_ERROR_BUSY;
			default:
				usbi_err(ctx, "could not claim interface %d: %s", iface, windows_error_str(0));
				return LIBUSB_ERROR_ACCESS;
			}
		}
	}
	usbi_dbg("claimed interface %d", iface);
	handle_priv->active_interface = iface;

	return LIBUSB_SUCCESS;
}

static int winusbx_release_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE winusb_handle;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	winusb_handle = handle_priv->interface_handle[iface].api_handle;
	if (!HANDLE_VALID(winusb_handle))
		return LIBUSB_ERROR_NOT_FOUND;

	WinUSBX[sub_api].Free(winusb_handle);
	handle_priv->interface_handle[iface].api_handle = INVALID_HANDLE_VALUE;

	return LIBUSB_SUCCESS;
}

/*
 * Return the first valid interface (of the same API type), for control transfers
 */
static int get_valid_interface(struct libusb_device_handle *dev_handle, int api_id)
{
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int i;

	if ((api_id < USB_API_WINUSBX) || (api_id > USB_API_HID)) {
		usbi_dbg("unsupported API ID");
		return -1;
	}

	for (i = 0; i < USB_MAXINTERFACES; i++) {
		if (HANDLE_VALID(handle_priv->interface_handle[i].dev_handle)
				&& HANDLE_VALID(handle_priv->interface_handle[i].api_handle)
				&& (priv->usb_interface[i].apib->id == api_id))
			return i;
	}

	return -1;
}

/*
 * Lookup interface by endpoint address. -1 if not found
 */
static int interface_by_endpoint(struct winusb_device_priv *priv,
	struct winusb_device_handle_priv *handle_priv, uint8_t endpoint_address)
{
	int i, j;

	for (i = 0; i < USB_MAXINTERFACES; i++) {
		if (!HANDLE_VALID(handle_priv->interface_handle[i].api_handle))
			continue;
		if (priv->usb_interface[i].endpoint == NULL)
			continue;
		for (j = 0; j < priv->usb_interface[i].nb_endpoints; j++) {
			if (priv->usb_interface[i].endpoint[j] == endpoint_address)
				return i;
		}
	}

	return -1;
}

static int winusbx_submit_control_transfer(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	PWINUSB_SETUP_PACKET setup = (PWINUSB_SETUP_PACKET)transfer->buffer;
	ULONG size;
	HANDLE winusb_handle;
	OVERLAPPED *overlapped;
	int current_interface;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	size = transfer->length - LIBUSB_CONTROL_SETUP_SIZE;

	// Windows places upper limits on the control transfer size
	// See: https://msdn.microsoft.com/en-us/library/windows/hardware/ff538112.aspx
	if (size > MAX_CTRL_BUFFER_LENGTH)
		return LIBUSB_ERROR_INVALID_PARAM;

	current_interface = get_valid_interface(transfer->dev_handle, USB_API_WINUSBX);
	if (current_interface < 0) {
		if (auto_claim(transfer, &current_interface, USB_API_WINUSBX) != LIBUSB_SUCCESS)
			return LIBUSB_ERROR_NOT_FOUND;
	}

	usbi_dbg("will use interface %d", current_interface);

	transfer_priv->handle = winusb_handle = handle_priv->interface_handle[current_interface].api_handle;
	overlapped = transfer_priv->pollable_fd.overlapped;

	// Sending of set configuration control requests from WinUSB creates issues
	if ((LIBUSB_REQ_TYPE(setup->RequestType) == LIBUSB_REQUEST_TYPE_STANDARD)
			&& (setup->Request == LIBUSB_REQUEST_SET_CONFIGURATION)) {
		if (setup->Value != priv->active_config) {
			usbi_warn(ctx, "cannot set configuration other than the default one");
			return LIBUSB_ERROR_INVALID_PARAM;
		}
		windows_force_sync_completion(overlapped, 0);
	} else {
		if (!WinUSBX[sub_api].ControlTransfer(winusb_handle, *setup, transfer->buffer + LIBUSB_CONTROL_SETUP_SIZE, size, NULL, overlapped)) {
			if (GetLastError() != ERROR_IO_PENDING) {
				usbi_warn(ctx, "ControlTransfer failed: %s", windows_error_str(0));
				return LIBUSB_ERROR_IO;
			}
		} else {
			windows_force_sync_completion(overlapped, size);
		}
	}

	transfer_priv->interface_number = (uint8_t)current_interface;

	return LIBUSB_SUCCESS;
}

static int winusbx_set_interface_altsetting(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE winusb_handle;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	if (altsetting > 255)
		return LIBUSB_ERROR_INVALID_PARAM;

	winusb_handle = handle_priv->interface_handle[iface].api_handle;
	if (!HANDLE_VALID(winusb_handle)) {
		usbi_err(ctx, "interface must be claimed first");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	if (!WinUSBX[sub_api].SetCurrentAlternateSetting(winusb_handle, (UCHAR)altsetting)) {
		usbi_err(ctx, "SetCurrentAlternateSetting failed: %s", windows_error_str(0));
		return LIBUSB_ERROR_IO;
	}

	return LIBUSB_SUCCESS;
}

static int winusbx_submit_iso_transfer(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	HANDLE winusb_handle;
	OVERLAPPED *overlapped;
	bool ret;
	int current_interface;
	int i;
	UINT offset;
	PKISO_CONTEXT iso_context;
	size_t iso_ctx_size;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	if ((sub_api != SUB_API_LIBUSBK) && (sub_api != SUB_API_LIBUSB0)) {
		// iso only supported on libusbk-based backends
		PRINT_UNSUPPORTED_API(submit_iso_transfer);
		return LIBUSB_ERROR_NOT_SUPPORTED;
	};

	current_interface = interface_by_endpoint(priv, handle_priv, transfer->endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cancelling transfer");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	usbi_dbg("matched endpoint %02X with interface %d", transfer->endpoint, current_interface);

	transfer_priv->handle = winusb_handle = handle_priv->interface_handle[current_interface].api_handle;
	overlapped = transfer_priv->pollable_fd.overlapped;

	iso_ctx_size = sizeof(KISO_CONTEXT) + (transfer->num_iso_packets * sizeof(KISO_PACKET));
	transfer_priv->iso_context = iso_context = calloc(1, iso_ctx_size);
	if (transfer_priv->iso_context == NULL)
		return LIBUSB_ERROR_NO_MEM;

	// start ASAP
	iso_context->StartFrame = 0;
	iso_context->NumberOfPackets = (SHORT)transfer->num_iso_packets;

	// convert the transfer packet lengths to iso_packet offsets
	offset = 0;
	for (i = 0; i < transfer->num_iso_packets; i++) {
		iso_context->IsoPackets[i].offset = offset;
		offset += transfer->iso_packet_desc[i].length;
	}

	if (IS_XFERIN(transfer)) {
		usbi_dbg("reading %d iso packets", transfer->num_iso_packets);
		ret = WinUSBX[sub_api].IsoReadPipe(winusb_handle, transfer->endpoint, transfer->buffer, transfer->length, overlapped, iso_context);
	} else {
		usbi_dbg("writing %d iso packets", transfer->num_iso_packets);
		ret = WinUSBX[sub_api].IsoWritePipe(winusb_handle, transfer->endpoint, transfer->buffer, transfer->length, overlapped, iso_context);
	}

	if (!ret) {
		if (GetLastError() != ERROR_IO_PENDING) {
			usbi_err(ctx, "IsoReadPipe/IsoWritePipe failed: %s", windows_error_str(0));
			return LIBUSB_ERROR_IO;
		}
	} else {
		windows_force_sync_completion(overlapped, (ULONG)transfer->length);
	}

	transfer_priv->interface_number = (uint8_t)current_interface;

	return LIBUSB_SUCCESS;
}

static int winusbx_submit_bulk_transfer(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	HANDLE winusb_handle;
	OVERLAPPED *overlapped;
	bool ret;
	int current_interface;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	current_interface = interface_by_endpoint(priv, handle_priv, transfer->endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cancelling transfer");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	usbi_dbg("matched endpoint %02X with interface %d", transfer->endpoint, current_interface);

	transfer_priv->handle = winusb_handle = handle_priv->interface_handle[current_interface].api_handle;
	overlapped = transfer_priv->pollable_fd.overlapped;

	if (IS_XFERIN(transfer)) {
		usbi_dbg("reading %d bytes", transfer->length);
		ret = WinUSBX[sub_api].ReadPipe(winusb_handle, transfer->endpoint, transfer->buffer, transfer->length, NULL, overlapped);
	} else {
		usbi_dbg("writing %d bytes", transfer->length);
		ret = WinUSBX[sub_api].WritePipe(winusb_handle, transfer->endpoint, transfer->buffer, transfer->length, NULL, overlapped);
	}

	if (!ret) {
		if (GetLastError() != ERROR_IO_PENDING) {
			usbi_err(ctx, "ReadPipe/WritePipe failed: %s", windows_error_str(0));
			return LIBUSB_ERROR_IO;
		}
	} else {
		windows_force_sync_completion(overlapped, (ULONG)transfer->length);
	}

	transfer_priv->interface_number = (uint8_t)current_interface;

	return LIBUSB_SUCCESS;
}

static int winusbx_clear_halt(int sub_api, struct libusb_device_handle *dev_handle, unsigned char endpoint)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE winusb_handle;
	int current_interface;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	current_interface = interface_by_endpoint(priv, handle_priv, endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cannot clear");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	usbi_dbg("matched endpoint %02X with interface %d", endpoint, current_interface);
	winusb_handle = handle_priv->interface_handle[current_interface].api_handle;

	if (!WinUSBX[sub_api].ResetPipe(winusb_handle, endpoint)) {
		usbi_err(ctx, "ResetPipe failed: %s", windows_error_str(0));
		return LIBUSB_ERROR_NO_DEVICE;
	}

	return LIBUSB_SUCCESS;
}

/*
 * from http://www.winvistatips.com/winusb-bugchecks-t335323.html (confirmed
 * through testing as well):
 * "You can not call WinUsb_AbortPipe on control pipe. You can possibly cancel
 * the control transfer using CancelIo"
 */
static int winusbx_abort_control(int sub_api, struct usbi_transfer *itransfer)
{
	// Cancelling of the I/O is done in the parent
	return LIBUSB_SUCCESS;
}

static int winusbx_abort_transfers(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	HANDLE handle;
	int current_interface;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	current_interface = transfer_priv->interface_number;
	if ((current_interface < 0) || (current_interface >= USB_MAXINTERFACES)) {
		usbi_err(ctx, "program assertion failed: invalid interface_number");
		return LIBUSB_ERROR_NOT_FOUND;
	}
	usbi_dbg("will use interface %d", current_interface);

	if (WinUSBX[sub_api].CancelIoEx_supported) {
		// Try to use CancelIoEx if available to cancel just a single transfer
		handle = handle_priv->interface_handle[current_interface].dev_handle;
		if (pCancelIoEx(handle, transfer_priv->pollable_fd.overlapped))
			return LIBUSB_SUCCESS;
		else if (GetLastError() == ERROR_NOT_FOUND)
			return LIBUSB_ERROR_NOT_FOUND;

		// Not every driver implements the necessary functionality for CancelIoEx
		usbi_warn(ctx, "CancelIoEx not supported for sub API %s", winusbx_driver_names[sub_api]);
		WinUSBX[sub_api].CancelIoEx_supported = false;
	}

	handle = handle_priv->interface_handle[current_interface].api_handle;
	if (!WinUSBX[sub_api].AbortPipe(handle, transfer->endpoint)) {
		usbi_err(ctx, "AbortPipe failed: %s", windows_error_str(0));
		return LIBUSB_ERROR_NO_DEVICE;
	}

	return LIBUSB_SUCCESS;
}

/*
 * from the "How to Use WinUSB to Communicate with a USB Device" Microsoft white paper
 * (http://www.microsoft.com/whdc/connect/usb/winusb_howto.mspx):
 * "WinUSB does not support host-initiated reset port and cycle port operations" and
 * IOCTL_INTERNAL_USB_CYCLE_PORT is only available in kernel mode and the
 * IOCTL_USB_HUB_CYCLE_PORT ioctl was removed from Vista => the best we can do is
 * cycle the pipes (and even then, the control pipe can not be reset using WinUSB)
 */
// TODO: (post hotplug): see if we can force eject the device and redetect it (reuse hotplug?)
static int winusbx_reset_device(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	HANDLE winusb_handle;
	int i, j;

	CHECK_WINUSBX_AVAILABLE(sub_api);

	// Reset any available pipe (except control)
	for (i = 0; i < USB_MAXINTERFACES; i++) {
		winusb_handle = handle_priv->interface_handle[i].api_handle;
		if (HANDLE_VALID(winusb_handle)) {
			for (j = 0; j < priv->usb_interface[i].nb_endpoints; j++) {
				usbi_dbg("resetting ep %02X", priv->usb_interface[i].endpoint[j]);
				if (!WinUSBX[sub_api].AbortPipe(winusb_handle, priv->usb_interface[i].endpoint[j]))
					usbi_err(ctx, "AbortPipe (pipe address %02X) failed: %s",
						priv->usb_interface[i].endpoint[j], windows_error_str(0));

				// FlushPipe seems to fail on OUT pipes
				if (IS_EPIN(priv->usb_interface[i].endpoint[j])
						&& (!WinUSBX[sub_api].FlushPipe(winusb_handle, priv->usb_interface[i].endpoint[j])))
					usbi_err(ctx, "FlushPipe (pipe address %02X) failed: %s",
						priv->usb_interface[i].endpoint[j], windows_error_str(0));

				if (!WinUSBX[sub_api].ResetPipe(winusb_handle, priv->usb_interface[i].endpoint[j]))
					usbi_err(ctx, "ResetPipe (pipe address %02X) failed: %s",
						priv->usb_interface[i].endpoint[j], windows_error_str(0));
			}
		}
	}

	// libusbK & libusb0 have the ability to issue an actual device reset
	if (WinUSBX[sub_api].ResetDevice != NULL) {
		winusb_handle = handle_priv->interface_handle[0].api_handle;
		if (HANDLE_VALID(winusb_handle))
			WinUSBX[sub_api].ResetDevice(winusb_handle);
	}

	return LIBUSB_SUCCESS;
}

static int winusbx_copy_transfer_data(int sub_api, struct usbi_transfer *itransfer, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	PKISO_CONTEXT iso_context;
	int i;

	if (transfer->type == LIBUSB_TRANSFER_TYPE_ISOCHRONOUS) {
		CHECK_WINUSBX_AVAILABLE(sub_api);

		// for isochronous, need to copy the individual iso packet actual_lengths and statuses
		if ((sub_api == SUB_API_LIBUSBK) || (sub_api == SUB_API_LIBUSB0)) {
			// iso only supported on libusbk-based backends for now
			iso_context = transfer_priv->iso_context;
			for (i = 0; i < transfer->num_iso_packets; i++) {
				transfer->iso_packet_desc[i].actual_length = iso_context->IsoPackets[i].actual_length;
				// TODO translate USDB_STATUS codes http://msdn.microsoft.com/en-us/library/ff539136(VS.85).aspx to libusb_transfer_status
				//transfer->iso_packet_desc[i].status = transfer_priv->iso_context->IsoPackets[i].status;
			}
		} else {
			// This should only occur if backend is not set correctly or other backend isoc is partially implemented
			PRINT_UNSUPPORTED_API(copy_transfer_data);
			return LIBUSB_ERROR_NOT_SUPPORTED;
		}
	}

	itransfer->transferred += io_size;
	return LIBUSB_TRANSFER_COMPLETED;
}

/*
 * Composite API functions
 */
static int composite_open(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int r = LIBUSB_ERROR_NOT_FOUND;
	uint8_t i;
	// SUB_API_MAX + 1 as the SUB_API_MAX pos is used to indicate availability of HID
	bool available[SUB_API_MAX + 1] = { 0 };

	for (i = 0; i < USB_MAXINTERFACES; i++) {
		switch (priv->usb_interface[i].apib->id) {
		case USB_API_WINUSBX:
			if (priv->usb_interface[i].sub_api != SUB_API_NOTSET) {
				available[priv->usb_interface[i].sub_api] = true;
			}
			break;
		case USB_API_HID:
			available[SUB_API_MAX] = true;
			break;
		default:
			break;
		}
	}

	for (i = 0; i < SUB_API_MAX; i++) { // WinUSB-like drivers
		if (available[i]) {
			r = usb_api_backend[USB_API_WINUSBX].open(i, dev_handle);
			if (r != LIBUSB_SUCCESS) {
				return r;
			}
		}
	}
/*
	if (available[SUB_API_MAX]) // HID driver
		r = hid_open(SUB_API_NOTSET, dev_handle);
*/
	return r;
}

static void composite_close(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	uint8_t i;
	// SUB_API_MAX + 1 as the SUB_API_MAX pos is used to indicate availability of HID
	bool available[SUB_API_MAX + 1] = { 0 };

	for (i = 0; i < USB_MAXINTERFACES; i++) {
		switch (priv->usb_interface[i].apib->id) {
		case USB_API_WINUSBX:
			if (priv->usb_interface[i].sub_api != SUB_API_NOTSET)
				available[priv->usb_interface[i].sub_api] = true;
			break;
		case USB_API_HID:
			available[SUB_API_MAX] = true;
			break;
		default:
			break;
		}
	}

	for (i = 0; i < SUB_API_MAX; i++) { // WinUSB-like drivers
		if (available[i])
			usb_api_backend[USB_API_WINUSBX].close(i, dev_handle);
	}
/*
	if (available[SUB_API_MAX]) // HID driver
		hid_close(SUB_API_NOTSET, dev_handle);
*/
}

static int composite_claim_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->usb_interface[iface].apib, claim_interface);

	return priv->usb_interface[iface].apib->
		claim_interface(priv->usb_interface[iface].sub_api, dev_handle, iface);
}

static int composite_set_interface_altsetting(int sub_api, struct libusb_device_handle *dev_handle, int iface, int altsetting)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->usb_interface[iface].apib, set_interface_altsetting);

	return priv->usb_interface[iface].apib->
		set_interface_altsetting(priv->usb_interface[iface].sub_api, dev_handle, iface, altsetting);
}

static int composite_release_interface(int sub_api, struct libusb_device_handle *dev_handle, int iface)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);

	CHECK_SUPPORTED_API(priv->usb_interface[iface].apib, release_interface);

	return priv->usb_interface[iface].apib->
		release_interface(priv->usb_interface[iface].sub_api, dev_handle, iface);
}

static int composite_submit_control_transfer(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	struct libusb_config_descriptor *conf_desc;
	WINUSB_SETUP_PACKET *setup = (WINUSB_SETUP_PACKET *)transfer->buffer;
	int iface, pass, r;

	// Interface shouldn't matter for control, but it does in practice, with Windows'
	// restrictions with regards to accessing HID keyboards and mice. Try to target
	// a specific interface first, if possible.
	switch (LIBUSB_REQ_RECIPIENT(setup->RequestType)) {
	case LIBUSB_RECIPIENT_INTERFACE:
		iface = setup->Index & 0xFF;
		break;
	case LIBUSB_RECIPIENT_ENDPOINT:
		r = libusb_get_active_config_descriptor(transfer->dev_handle->dev, &conf_desc);
		if (r == LIBUSB_SUCCESS) {
			iface = get_interface_by_endpoint(conf_desc, (setup->Index & 0xFF));
			libusb_free_config_descriptor(conf_desc);
			break;
		}
		// Fall through if not able to determine interface
	default:
		iface = -1;
		break;
	}

	// Try and target a specific interface if the control setup indicates such
	if ((iface >= 0) && (iface < USB_MAXINTERFACES)) {
		usbi_dbg("attempting control transfer targeted to interface %d", iface);
		if ((priv->usb_interface[iface].path != NULL)
				&& (priv->usb_interface[iface].apib->submit_control_transfer != NULL)) {
			r = priv->usb_interface[iface].apib->submit_control_transfer(priv->usb_interface[iface].sub_api, itransfer);
			if (r == LIBUSB_SUCCESS)
				return r;
		}
	}

	// Either not targeted to a specific interface or no luck in doing so.
	// Try a 2 pass approach with all interfaces.
	for (pass = 0; pass < 2; pass++) {
		for (iface = 0; iface < USB_MAXINTERFACES; iface++) {
			if ((priv->usb_interface[iface].path != NULL)
					&& (priv->usb_interface[iface].apib->submit_control_transfer != NULL)) {
				if ((pass == 0) && (priv->usb_interface[iface].restricted_functionality)) {
					usbi_dbg("trying to skip restricted interface #%d (HID keyboard or mouse?)", iface);
					continue;
				}
				usbi_dbg("using interface %d", iface);
				r = priv->usb_interface[iface].apib->submit_control_transfer(priv->usb_interface[iface].sub_api, itransfer);
				// If not supported on this API, it may be supported on another, so don't give up yet!!
				if (r == LIBUSB_ERROR_NOT_SUPPORTED)
					continue;
				return r;
			}
		}
	}
	usbi_err(ctx, "no libusb supported interfaces to complete request");
	return LIBUSB_ERROR_NOT_FOUND;
}

static int composite_submit_bulk_transfer(int sub_api, struct usbi_transfer *itransfer) {
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface;

	current_interface = interface_by_endpoint(priv, handle_priv, transfer->endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cancelling transfer");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, submit_bulk_transfer);

	return priv->usb_interface[current_interface].apib->
		submit_bulk_transfer(priv->usb_interface[current_interface].sub_api, itransfer);
}

static int composite_submit_iso_transfer(int sub_api, struct usbi_transfer *itransfer) {
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct libusb_context *ctx = DEVICE_CTX(transfer->dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(transfer->dev_handle);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface;

	current_interface = interface_by_endpoint(priv, handle_priv, transfer->endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cancelling transfer");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, submit_iso_transfer);

	return priv->usb_interface[current_interface].apib->
		submit_iso_transfer(priv->usb_interface[current_interface].sub_api, itransfer);
}

static int composite_clear_halt(int sub_api, struct libusb_device_handle *dev_handle, unsigned char endpoint)
{
	struct libusb_context *ctx = DEVICE_CTX(dev_handle->dev);
	struct winusb_device_handle_priv *handle_priv = _device_handle_priv(dev_handle);
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int current_interface;

	current_interface = interface_by_endpoint(priv, handle_priv, endpoint);
	if (current_interface < 0) {
		usbi_err(ctx, "unable to match endpoint to an open interface - cannot clear");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, clear_halt);

	return priv->usb_interface[current_interface].apib->
		clear_halt(priv->usb_interface[current_interface].sub_api, dev_handle, endpoint);
}

static int composite_abort_control(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface = transfer_priv->interface_number;

	if ((current_interface < 0) || (current_interface >= USB_MAXINTERFACES)) {
		usbi_err(TRANSFER_CTX(transfer), "program assertion failed: invalid interface_number");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, abort_control);

	return priv->usb_interface[current_interface].apib->
		abort_control(priv->usb_interface[current_interface].sub_api, itransfer);
}

static int composite_abort_transfers(int sub_api, struct usbi_transfer *itransfer)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface = transfer_priv->interface_number;

	if ((current_interface < 0) || (current_interface >= USB_MAXINTERFACES)) {
		usbi_err(TRANSFER_CTX(transfer), "program assertion failed: invalid interface_number");
		return LIBUSB_ERROR_NOT_FOUND;
	}

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, abort_transfers);

	return priv->usb_interface[current_interface].apib->
		abort_transfers(priv->usb_interface[current_interface].sub_api, itransfer);
}

static int composite_reset_device(int sub_api, struct libusb_device_handle *dev_handle)
{
	struct winusb_device_priv *priv = _device_priv(dev_handle->dev);
	int r;
	uint8_t i;
	bool available[SUB_API_MAX];

	for (i = 0; i < SUB_API_MAX; i++)
		available[i] = false;

	for (i = 0; i < USB_MAXINTERFACES; i++) {
		if ((priv->usb_interface[i].apib->id == USB_API_WINUSBX)
				&& (priv->usb_interface[i].sub_api != SUB_API_NOTSET))
			available[priv->usb_interface[i].sub_api] = true;
	}

	for (i = 0; i < SUB_API_MAX; i++) {
		if (available[i]) {
			r = usb_api_backend[USB_API_WINUSBX].reset_device(i, dev_handle);
			if (r != LIBUSB_SUCCESS)
				return r;
		}
	}

	return LIBUSB_SUCCESS;
}

static int composite_copy_transfer_data(int sub_api, struct usbi_transfer *itransfer, uint32_t io_size)
{
	struct libusb_transfer *transfer = USBI_TRANSFER_TO_LIBUSB_TRANSFER(itransfer);
	struct winusb_transfer_priv *transfer_priv = usbi_transfer_get_os_priv(itransfer);
	struct winusb_device_priv *priv = _device_priv(transfer->dev_handle->dev);
	int current_interface = transfer_priv->interface_number;

	CHECK_SUPPORTED_API(priv->usb_interface[current_interface].apib, copy_transfer_data);

	return priv->usb_interface[current_interface].apib->
		copy_transfer_data(priv->usb_interface[current_interface].sub_api, itransfer, io_size);
}
