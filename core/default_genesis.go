// Copyright 2014 The go-ethereum Authors && Copyright 2015 go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"compress/gzip"
	"encoding/base64"
	"io"
	"strings"
)

func NewDefaultGenesisReader() (io.Reader, error) {
	return gzip.NewReader(base64.NewDecoder(base64.StdEncoding, strings.NewReader(defaultGenesisBlock)))
}

const defaultGenesisBlock = "H4sICGn59VUAA2dlbmVzaXMuanNvbgCtUctqwzAQPDdfYXzOYSWvVlLOPfTQn1jJUmOwnRCr4BL871X8KKVQMLSrg9jXzGh0PxQ5yv7S+1CeihJGKZCURIEBHUYpyuMykpouDIm76zIGW/3Kt9CnFx7Oa+OPseGGMd34mRMvsEhakqBAjiJaXc25ygcpUr0tvfHw2nRNWnZEZczWqZsYG//epo8V7wdd14znf3+DvzS942F11lZ18OzA1xqVssYQRG0kexFMBCTvrA3Gfknmtr34vHqf07kEo3MWI/jgKimVllo5yZqtB4teVAHJOlRUPSjvRem45fVnxe9qi+n4nUI8bNVCAZmstnZ1hHyDRsOGhQ0IURph5X6KMhM8ZWQPWkUWIhrlaq6YHWciICmJOYKh7EX+3v3iN+Dd1u4ELqfZkOkwHT4BD678BCIDAAA="