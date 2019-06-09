#pragma once

#include "windows_common.h"

#include <pshpack1.h>

typedef struct USB_DEVICE_DESCRIPTOR {
	UCHAR  bLength;
	UCHAR  bDescriptorType;
	USHORT bcdUSB;
	UCHAR  bDeviceClass;
	UCHAR  bDeviceSubClass;
	UCHAR  bDeviceProtocol;
	UCHAR  bMaxPacketSize0;
	USHORT idVendor;
	USHORT idProduct;
	USHORT bcdDevice;
	UCHAR  iManufacturer;
	UCHAR  iProduct;
	UCHAR  iSerialNumber;
	UCHAR  bNumConfigurations;
} USB_DEVICE_DESCRIPTOR, *PUSB_DEVICE_DESCRIPTOR;

typedef struct USB_CONFIGURATION_DESCRIPTOR {
	UCHAR  bLength;
	UCHAR  bDescriptorType;
	USHORT wTotalLength;
	UCHAR  bNumInterfaces;
	UCHAR  bConfigurationValue;
	UCHAR  iConfiguration;
	UCHAR  bmAttributes;
	UCHAR  MaxPower;
} USB_CONFIGURATION_DESCRIPTOR, *PUSB_CONFIGURATION_DESCRIPTOR;

#include <poppack.h>

#define MAX_DEVICE_ID_LEN	200

typedef struct USB_DK_DEVICE_ID {
	WCHAR DeviceID[MAX_DEVICE_ID_LEN];
	WCHAR InstanceID[MAX_DEVICE_ID_LEN];
} USB_DK_DEVICE_ID, *PUSB_DK_DEVICE_ID;

typedef struct USB_DK_DEVICE_INFO {
	USB_DK_DEVICE_ID ID;
	ULONG64 FilterID;
	ULONG64 Port;
	ULONG64 Speed;
	USB_DEVICE_DESCRIPTOR DeviceDescriptor;
} USB_DK_DEVICE_INFO, *PUSB_DK_DEVICE_INFO;

typedef struct USB_DK_ISO_TRANSFER_RESULT {
	ULONG64 ActualLength;
	ULONG64 TransferResult;
} USB_DK_ISO_TRANSFER_RESULT, *PUSB_DK_ISO_TRANSFER_RESULT;

typedef struct USB_DK_GEN_TRANSFER_RESULT {
	ULONG64 BytesTransferred;
	ULONG64 UsbdStatus; // USBD_STATUS code
} USB_DK_GEN_TRANSFER_RESULT, *PUSB_DK_GEN_TRANSFER_RESULT;

typedef struct USB_DK_TRANSFER_RESULT {
	USB_DK_GEN_TRANSFER_RESULT GenResult;
	PVOID64 IsochronousResultsArray; // array of USB_DK_ISO_TRANSFER_RESULT
} USB_DK_TRANSFER_RESULT, *PUSB_DK_TRANSFER_RESULT;

typedef struct USB_DK_TRANSFER_REQUEST {
	ULONG64 EndpointAddress;
	PVOID64 Buffer;
	ULONG64 BufferLength;
	ULONG64 TransferType;
	ULONG64 IsochronousPacketsArraySize;
	PVOID64 IsochronousPacketsArray;
	USB_DK_TRANSFER_RESULT Result;
} USB_DK_TRANSFER_REQUEST, *PUSB_DK_TRANSFER_REQUEST;

struct usbdk_device_priv {
	USB_DK_DEVICE_INFO info;
	PUSB_CONFIGURATION_DESCRIPTOR *config_descriptors;
	HANDLE redirector_handle;
	HANDLE system_handle;
	uint8_t active_configuration;
};

struct winusb_device_priv {
	bool initialized;
	bool root_hub;
	uint8_t active_config;
	uint8_t depth; // distance to HCD
	const struct windows_usb_api_backend *apib;
	char *dev_id;
	char *path;  // device interface path
	int sub_api; // for WinUSB-like APIs
	struct {
		char *path; // each interface needs a device interface path,
		const struct windows_usb_api_backend *apib; // an API backend (multiple drivers support),
		int sub_api;
		int8_t nb_endpoints; // and a set of endpoint addresses (USB_MAXENDPOINTS)
		uint8_t *endpoint;
		bool restricted_functionality;  // indicates if the interface functionality is restricted
						// by Windows (eg. HID keyboards or mice cannot do R/W)
	} usb_interface[USB_MAXINTERFACES];
	struct hid_device_priv *hid;
	USB_DEVICE_DESCRIPTOR dev_descriptor;
	PUSB_CONFIGURATION_DESCRIPTOR *config_descriptor; // list of pointers to the cached config descriptors
};

struct usbdk_device_handle_priv {
	// Not currently used
	char dummy;
};

struct winusb_device_handle_priv {
	int active_interface;
	struct {
		HANDLE dev_handle; // WinUSB needs an extra handle for the file
		HANDLE api_handle; // used by the API to communicate with the device
	} interface_handle[USB_MAXINTERFACES];
	int autoclaim_count[USB_MAXINTERFACES]; // For auto-release
};

struct usbdk_transfer_priv {
	USB_DK_TRANSFER_REQUEST request;
	struct winfd pollable_fd;
	HANDLE system_handle;
	PULONG64 IsochronousPacketsArray;
	PUSB_DK_ISO_TRANSFER_RESULT IsochronousResultsArray;
};

struct winusb_transfer_priv {
	struct winfd pollable_fd;
	HANDLE handle;
	uint8_t interface_number;
	uint8_t *hid_buffer; // 1 byte extended data buffer, required for HID
	uint8_t *hid_dest;   // transfer buffer destination, required for HID
	size_t hid_expected_size;
	void *iso_context;
};
