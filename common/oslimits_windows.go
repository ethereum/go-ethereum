package common

import (
)

func MaxOpenFileLimit() int {
	// From https://msdn.microsoft.com/en-us/library/kdfaxaay.aspx
        return 512
}
