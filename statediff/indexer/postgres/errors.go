// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package postgres

import (
	"fmt"
)

const (
	BeginTransactionFailedMsg = "failed to begin transaction"
	DbConnectionFailedMsg     = "db connection failed"
	DeleteQueryFailedMsg      = "delete query failed"
	InsertQueryFailedMsg      = "insert query failed"
	SettingNodeFailedMsg      = "unable to set db node"
)

func ErrBeginTransactionFailed(beginErr error) error {
	return formatError(BeginTransactionFailedMsg, beginErr.Error())
}

func ErrDBConnectionFailed(connectErr error) error {
	return formatError(DbConnectionFailedMsg, connectErr.Error())
}

func ErrDBDeleteFailed(deleteErr error) error {
	return formatError(DeleteQueryFailedMsg, deleteErr.Error())
}

func ErrDBInsertFailed(insertErr error) error {
	return formatError(InsertQueryFailedMsg, insertErr.Error())
}

func ErrUnableToSetNode(setErr error) error {
	return formatError(SettingNodeFailedMsg, setErr.Error())
}

func formatError(msg, err string) error {
	return fmt.Errorf("%s: %s", msg, err)
}
