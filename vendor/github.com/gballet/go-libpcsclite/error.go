// BSD 3-Clause License
//
// Copyright (c) 2019, Guillaume Ballet
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
//
// * Neither the name of the copyright holder nor the names of its
//   contributors may be used to endorse or promote products derived from
//   this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package pcsc

import "fmt"

type ErrorCode uint32

const (
	SCardSuccess                   ErrorCode = 0x00000000 /* No error was encountered. */
	ErrSCardInternal                         = 0x80100001 /* An internal consistency check failed. */
	ErrSCardCancelled                        = 0x80100002 /* The action was cancelled by an SCardCancel request. */
	ErrSCardInvalidHandle                    = 0x80100003 /* The supplied handle was invalid. */
	ErrSCardInvalidParameter                 = 0x80100004 /* One or more of the supplied parameters could not be properly interpreted. */
	ErrSCardInvalidTarget                    = 0x80100005 /* Registry startup information is missing or invalid. */
	ErrSCardNoMemory                         = 0x80100006 /* Not enough memory available to complete this command. */
	ErrSCardWaitedTooLong                    = 0x80100007 /* An internal consistency timer has expired. */
	ErrSCardInsufficientBuffer               = 0x80100008 /* The data buffer to receive returned data is too small for the returned data. */
	ErrScardUnknownReader                    = 0x80100009 /* The specified reader name is not recognized. */
	ErrSCardTimeout                          = 0x8010000A /* The user-specified timeout value has expired. */
	ErrSCardSharingViolation                 = 0x8010000B /* The smart card cannot be accessed because of other connections outstanding. */
	ErrSCardNoSmartCard                      = 0x8010000C /* The operation requires a Smart Card, but no Smart Card is currently in the device. */
	ErrSCardUnknownCard                      = 0x8010000D /* The specified smart card name is not recognized. */
	ErrSCardCannotDispose                    = 0x8010000E /* The system could not dispose of the media in the requested manner. */
	ErrSCardProtoMismatch                    = 0x8010000F /* The requested protocols are incompatible with the protocol currently in use with the smart card. */
	ErrSCardNotReady                         = 0x80100010 /* The reader or smart card is not ready to accept commands. */
	ErrSCardInvalidValue                     = 0x80100011 /* One or more of the supplied parameters values could not be properly interpreted. */
	ErrSCardSystemCancelled                  = 0x80100012 /* The action was cancelled by the system, presumably to log off or shut down. */
	ErrSCardCommError                        = 0x80100013 /* An internal communications error has been detected. */
	ErrScardUnknownError                     = 0x80100014 /* An internal error has been detected, but the source is unknown. */
	ErrSCardInvalidATR                       = 0x80100015 /* An ATR obtained from the registry is not a valid ATR string. */
	ErrSCardNotTransacted                    = 0x80100016 /* An attempt was made to end a non-existent transaction. */
	ErrSCardReaderUnavailable                = 0x80100017 /* The specified reader is not currently available for use. */
	ErrSCardShutdown                         = 0x80100018 /* The operation has been aborted to allow the server application to exit. */
	ErrSCardPCITooSmall                      = 0x80100019 /* The PCI Receive buffer was too small. */
	ErrSCardReaderUnsupported                = 0x8010001A /* The reader driver does not meet minimal requirements for support. */
	ErrSCardDuplicateReader                  = 0x8010001B /* The reader driver did not produce a unique reader name. */
	ErrSCardCardUnsupported                  = 0x8010001C /* The smart card does not meet minimal requirements for support. */
	ErrScardNoService                        = 0x8010001D /* The Smart card resource manager is not running. */
	ErrSCardServiceStopped                   = 0x8010001E /* The Smart card resource manager has shut down. */
	ErrSCardUnexpected                       = 0x8010001F /* An unexpected card error has occurred. */
	ErrSCardUnsupportedFeature               = 0x8010001F /* This smart card does not support the requested feature. */
	ErrSCardICCInstallation                  = 0x80100020 /* No primary provider can be found for the smart card. */
	ErrSCardICCCreateOrder                   = 0x80100021 /* The requested order of object creation is not supported. */
	ErrSCardDirNotFound                      = 0x80100023 /* The identified directory does not exist in the smart card. */
	ErrSCardFileNotFound                     = 0x80100024 /* The identified file does not exist in the smart card. */
	ErrSCardNoDir                            = 0x80100025 /* The supplied path does not represent a smart card directory. */
	ErrSCardNoFile                           = 0x80100026 /* The supplied path does not represent a smart card file. */
	ErrScardNoAccess                         = 0x80100027 /* Access is denied to this file. */
	ErrSCardWriteTooMany                     = 0x80100028 /* The smart card does not have enough memory to store the information. */
	ErrSCardBadSeek                          = 0x80100029 /* There was an error trying to set the smart card file object pointer. */
	ErrSCardInvalidCHV                       = 0x8010002A /* The supplied PIN is incorrect. */
	ErrSCardUnknownResMNG                    = 0x8010002B /* An unrecognized error code was returned from a layered component. */
	ErrSCardNoSuchCertificate                = 0x8010002C /* The requested certificate does not exist. */
	ErrSCardCertificateUnavailable           = 0x8010002D /* The requested certificate could not be obtained. */
	ErrSCardNoReadersAvailable               = 0x8010002E /* Cannot find a smart card reader. */
	ErrSCardCommDataLost                     = 0x8010002F /* A communications error with the smart card has been detected. Retry the operation. */
	ErrScardNoKeyContainer                   = 0x80100030 /* The requested key container does not exist on the smart card. */
	ErrSCardServerTooBusy                    = 0x80100031 /* The Smart Card Resource Manager is too busy to complete this operation. */
	ErrSCardUnsupportedCard                  = 0x80100065 /* The reader cannot communicate with the card, due to ATR string configuration conflicts. */
	ErrSCardUnresponsiveCard                 = 0x80100066 /* The smart card is not responding to a reset. */
	ErrSCardUnpoweredCard                    = 0x80100067 /* Power has been removed from the smart card, so that further communication is not possible. */
	ErrSCardResetCard                        = 0x80100068 /* The smart card has been reset, so any shared state information is invalid. */
	ErrSCardRemovedCard                      = 0x80100069 /* The smart card has been removed, so further communication is not possible. */
	ErrSCardSecurityViolation                = 0x8010006A /* Access was denied because of a security violation. */
	ErrSCardWrongCHV                         = 0x8010006B /* The card cannot be accessed because the wrong PIN was presented. */
	ErrSCardCHVBlocked                       = 0x8010006C /* The card cannot be accessed because the maximum number of PIN entry attempts has been reached. */
	ErrSCardEOF                              = 0x8010006D /* The end of the smart card file has been reached. */
	ErrSCardCancelledByUser                  = 0x8010006E /* The user pressed "Cancel" on a Smart Card Selection Dialog. */
	ErrSCardCardNotAuthenticated             = 0x8010006F /* No PIN was presented to the smart card. */
)

// Code returns the error code, with an uint32 type to be used in PutUInt32
func (code ErrorCode) Code() uint32 {
	return uint32(code)
}

func (code ErrorCode) Error() error {
	switch code {
	case SCardSuccess:
		return fmt.Errorf("Command successful")

	case ErrSCardInternal:
		return fmt.Errorf("Internal error")

	case ErrSCardCancelled:
		return fmt.Errorf("Command cancelled")

	case ErrSCardInvalidHandle:
		return fmt.Errorf("Invalid handle")

	case ErrSCardInvalidParameter:
		return fmt.Errorf("Invalid parameter given")

	case ErrSCardInvalidTarget:
		return fmt.Errorf("Invalid target given")

	case ErrSCardNoMemory:
		return fmt.Errorf("Not enough memory")

	case ErrSCardWaitedTooLong:
		return fmt.Errorf("Waited too long")

	case ErrSCardInsufficientBuffer:
		return fmt.Errorf("Insufficient buffer")

	case ErrScardUnknownReader:
		return fmt.Errorf("Unknown reader specified")

	case ErrSCardTimeout:
		return fmt.Errorf("Command timeout")

	case ErrSCardSharingViolation:
		return fmt.Errorf("Sharing violation")

	case ErrSCardNoSmartCard:
		return fmt.Errorf("No smart card inserted")

	case ErrSCardUnknownCard:
		return fmt.Errorf("Unknown card")

	case ErrSCardCannotDispose:
		return fmt.Errorf("Cannot dispose handle")

	case ErrSCardProtoMismatch:
		return fmt.Errorf("Card protocol mismatch")

	case ErrSCardNotReady:
		return fmt.Errorf("Subsystem not ready")

	case ErrSCardInvalidValue:
		return fmt.Errorf("Invalid value given")

	case ErrSCardSystemCancelled:
		return fmt.Errorf("System cancelled")

	case ErrSCardCommError:
		return fmt.Errorf("RPC transport error")

	case ErrScardUnknownError:
		return fmt.Errorf("Unknown error")

	case ErrSCardInvalidATR:
		return fmt.Errorf("Invalid ATR")

	case ErrSCardNotTransacted:
		return fmt.Errorf("Transaction failed")

	case ErrSCardReaderUnavailable:
		return fmt.Errorf("Reader is unavailable")

	/* case SCARD_P_SHUTDOWN: */
	case ErrSCardPCITooSmall:
		return fmt.Errorf("PCI struct too small")

	case ErrSCardReaderUnsupported:
		return fmt.Errorf("Reader is unsupported")

	case ErrSCardDuplicateReader:
		return fmt.Errorf("Reader already exists")

	case ErrSCardCardUnsupported:
		return fmt.Errorf("Card is unsupported")

	case ErrScardNoService:
		return fmt.Errorf("Service not available")

	case ErrSCardServiceStopped:
		return fmt.Errorf("Service was stopped")

	/* case SCARD_E_UNEXPECTED: */
	/* case SCARD_E_ICC_CREATEORDER: */
	/* case SCARD_E_UNSUPPORTED_FEATURE: */
	/* case SCARD_E_DIR_NOT_FOUND: */
	/* case SCARD_E_NO_DIR: */
	/* case SCARD_E_NO_FILE: */
	/* case SCARD_E_NO_ACCESS: */
	/* case SCARD_E_WRITE_TOO_MANY: */
	/* case SCARD_E_BAD_SEEK: */
	/* case SCARD_E_INVALID_CHV: */
	/* case SCARD_E_UNKNOWN_RES_MNG: */
	/* case SCARD_E_NO_SUCH_CERTIFICATE: */
	/* case SCARD_E_CERTIFICATE_UNAVAILABLE: */
	case ErrSCardNoReadersAvailable:
		return fmt.Errorf("Cannot find a smart card reader")

	/* case SCARD_E_COMM_DATA_LOST: */
	/* case SCARD_E_NO_KEY_CONTAINER: */
	/* case SCARD_E_SERVER_TOO_BUSY: */
	case ErrSCardUnsupportedCard:
		return fmt.Errorf("Card is not supported")

	case ErrSCardUnresponsiveCard:
		return fmt.Errorf("Card is unresponsive")

	case ErrSCardUnpoweredCard:
		return fmt.Errorf("Card is unpowered")

	case ErrSCardResetCard:
		return fmt.Errorf("Card was reset")

	case ErrSCardRemovedCard:
		return fmt.Errorf("Card was removed")

	/* case SCARD_W_SECURITY_VIOLATION: */
	/* case SCARD_W_WRONG_CHV: */
	/* case SCARD_W_CHV_BLOCKED: */
	/* case SCARD_W_EOF: */
	/* case SCARD_W_CANCELLED_BY_USER: */
	/* case SCARD_W_CARD_NOT_AUTHENTICATED: */

	case ErrSCardUnsupportedFeature:
		return fmt.Errorf("Feature not supported")

	default:
		return fmt.Errorf("unknown error: %08x", code)
	}
}
