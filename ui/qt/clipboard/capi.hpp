#pragma once

#include "clipboard.hpp"

typedef void Clipboard_;

Clipboard_ *initClipboard()
{
	Clipboard *clipboard = new(Clipboard);
	return static_cast<Clipboard_*>(clipboard);
}
