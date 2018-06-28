package controllers

import (
	"html/template"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/api/http/messages"
	"github.com/ethereum/go-ethereum/swarm/api/http/views"
	l "github.com/ethereum/go-ethereum/swarm/log"
)

type Controller struct {
	api *api.API

	ControllerHandler
}

type ControllerHandler interface {
	Get(w http.ResponseWriter, r *messages.Request)
	Respond(w http.ResponseWriter, req *messages.Request, msg string, code int)
}

//Respond is used to show an HTML page to a client.
//If there is an `Accept` header of `application/json`, JSON will be returned instead
//The function just takes a string message which will be displayed in the error page.
//The code is used to evaluate which template will be displayed
//(and return the correct HTTP status code)
func (controller *Controller) Respond(w http.ResponseWriter, req *messages.Request, msg string, code int) {
	//additionalMessage := ValidateCaseErrors(req)
	//additionalMessage := ValidateCaseErrors(req)
	additionalMessage := ""
	switch code {
	case http.StatusInternalServerError:
		log.Output(msg, log.LvlError, l.CallDepth, "ruid", req.Ruid, "code", code)
	default:
		log.Output(msg, log.LvlDebug, l.CallDepth, "ruid", req.Ruid, "code", code)
	}

	if code >= 400 {
		w.Header().Del("Cache-Control") //avoid sending cache headers for errors!
		w.Header().Del("ETag")
	}

	respond(w, &req.Request, &messages.ResponseParams{
		Code:      code,
		Msg:       msg,
		Details:   template.HTML(additionalMessage),
		Timestamp: time.Now().Format(time.RFC1123),
		Template:  views.GetTemplate(code),
	})
}

//evaluate if client accepts html or json response
func respond(w http.ResponseWriter, r *http.Request, params *messages.ResponseParams) {
	w.WriteHeader(params.Code)
	if r.Header.Get("Accept") == "application/json" {
		//	respondJSON(w, params)
	} else {
		//	respondHTML(w, params)
	}
}
