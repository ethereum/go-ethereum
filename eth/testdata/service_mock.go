package main

import (
	"github.com/ethereum/go-ethereum/eth"
)

type Service struct {

}

func (s *Service) Namespace() string {
	return "test"
}

func (s *Service) Version() string {
	return "1.0.0"
}

type API struct {
	eth *eth.Ethereum
}

func (s *Service) Service(ethereum *eth.Ethereum) interface{} {
	return &API{eth: ethereum}
}

