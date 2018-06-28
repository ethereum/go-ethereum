package controllers

import (
	"net/http"

	"github.com/ethereum/go-ethereum/swarm/api/http/request"
)

type Controller struct {
	ControllerHandler
}

type ControllerHandler interface {
	Get(w http.ResponseWriter, r *request.Request)
}
