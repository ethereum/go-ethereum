// Copyright 2015 Google Inc.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid

import (
	"errors"
	"fmt"
)

// Scan implements sql.Scanner so UUIDs can be read from databases transparently
// Currently, database types that map to string and []byte are supported. Please
// consult database-specific driver documentation for matching types.
func (uuid *UUID) Scan(src interface{}) error {
	switch src.(type) {
	case string:
		// see uuid.Parse for required string format
		parsed := Parse(src.(string))

		if parsed == nil {
			return errors.New("Scan: invalid UUID format")
		}

		*uuid = parsed
	case []byte:
		// assumes a simple slice of bytes, just check validity and store
		u := UUID(src.([]byte))

		if u.Variant() == Invalid {
			return errors.New("Scan: invalid UUID format")
		}

		*uuid = u
	default:
		return fmt.Errorf("Scan: unable to scan type %T into UUID", src)
	}

	return nil
}
