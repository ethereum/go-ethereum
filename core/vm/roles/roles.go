//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package roles

import "fmt"

type Role byte

const (
	SenderDeployment    = 0xA0
	SenderValidation    = 0xA1
	PaymasterValidation = 0xA2
	SenderExecution     = 0xA3
	PaymasterPostOp     = 0xA4
)

func (r Role) String() string {
	if s := roleToString[r]; s != "" {
		return s
	}
	return fmt.Sprintf("role %#x not defined", int(r))
}

var roleToString = map[Role]string{
	SenderDeployment:    "SENDER_DEPLOYMENT",
	SenderValidation:    "SENDER_VALIDATION",
	PaymasterValidation: "PAYMASTER_VALIDATION",
	SenderExecution:     "SENDER_EXECUTION",
	PaymasterPostOp:     "PAYMASTER_POST_OP",
}

// var stringToRole = map[string]Role{
// 	"SENDER_DEPLOYMENT":    SenderDeployment,
// 	"SENDER_VALIDATION":    SenderValidation,
// 	"PAYMASTER_VALIDATION": PaymasterValidation,
// 	"SENDER_EXECUTION":     SenderExecution,
// 	"PAYMASTER_POST_OP":    PaymasterPostOp,
// }
