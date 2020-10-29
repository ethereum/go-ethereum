// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
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

package les

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/lotterybook"
	"github.com/ethereum/go-ethereum/les/lespay/payment/lotterypmt"
	"github.com/ethereum/go-ethereum/log"
)

// PaymentRobot is the testing tool for automatic payment cycle.
type PaymentRobot struct {
	sender   *lotterypmt.PaymentSender
	receiver common.Address
	close    chan struct{}
}

func NewPaymentRobot(manager *lotterypmt.PaymentSender, receiver common.Address, close chan struct{}) *PaymentRobot {
	return &PaymentRobot{
		sender:   manager,
		receiver: receiver,
		close:    close,
	}
}

type statistic struct {
	lotteries      uint64 // The total lotteries submitted
	paid           uint64 // The total paid times
	paidAmount     uint64 // The total paid amount
	multiLotteries uint64 // The total times we pay with multi-lotteries
	errors         uint64 // The total number of errors
}

func (s statistic) log() string {
	var msg string
	msg += "Payment robot stats\n\t"
	msg += fmt.Sprintf("Total lotteries: %d\n\t", s.lotteries)
	msg += fmt.Sprintf("Total paid made: %d\n\t", s.paid)
	msg += fmt.Sprintf("Total paid amount: %d\n\t", s.paidAmount)
	msg += fmt.Sprintf("Multi-lotteries: %d\n\t", s.multiLotteries)
	msg += fmt.Sprintf("Total errors: %d\n\t", s.errors)
	return msg
}

func (robot *PaymentRobot) Run(sendFn func(proofOfPayment []byte, identity string) error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	logTicker := time.NewTicker(time.Minute)
	defer logTicker.Stop()

	var (
		err     error
		deposit chan *lotterybook.LotteryEvent
		stat    statistic
	)
	depositFn := func() {
		if deposit != nil {
			log.Error("Depositing, skip new operation")
			return
		}
		list, amount := []common.Address{robot.receiver}, []uint64{100}
		if rand.Intn(2) > 0 {
			list = append(list, common.HexToAddress("0xdeadbeef"))
			amount = append(amount, uint64(rand.Intn(100)+50))
		}
		deposit, err = robot.sender.Deposit(list, amount, 60, nil)
		if err != nil {
			log.Error("Failed to deposit", "err", err)
			return
		}
		stat.lotteries += 1
	}
	depositFn()

	for {
		select {
		case <-ticker.C:
			if deposit != nil {
				continue
			}
			amount := uint64(3 + rand.Intn(10))
			proofOfPayments, err := robot.sender.Pay(robot.receiver, amount)
			if err != nil {
				if err == lotterybook.ErrNotEnoughDeposit {
					depositFn()
					continue
				}
				stat.errors += 1
				log.Error("Failed to pay", "error", err)
				continue
			}
			for _, proofOfPayment := range proofOfPayments {
				sendFn(proofOfPayment, lotterypmt.Identity)
			}
			if len(proofOfPayments) > 1 {
				stat.multiLotteries += 1
			}
			stat.paid += 1
			stat.paidAmount += amount

		case <-deposit:
			deposit = nil
			log.Info("Deposit finished")

		case <-logTicker.C:
			fmt.Println(stat.log())
			fmt.Println(robot.sender.DebugInspect())
			cost, _ := robot.sender.AverageCost().Float64()
			fmt.Printf("Average cost %.6f ether\n", cost)

		case <-robot.close:
			return
		}
	}
}
