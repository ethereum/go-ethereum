// Copyright 2015 The go-ethereum Authors
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

package ecp

import (
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type request struct {
	service string        // registered service name
	method  string        // callback on service
	args    []interface{} // arguments for callback
}

type callback struct {
	method reflect.Method // callback
	args   []reflect.Type // input argument types
	errPos int            // err return idx, of -1 when method cannot return error
}

type service struct {
	name      string        // name for service
	rcvr      reflect.Value // receiver of methods for the service
	typ       reflect.Type  // receiver type
	callbacks callbacks     // registered handlers
}

type serviceRegistry map[string]*service
type codecCollection map[ServerCodec]interface{}
type callbacks map[string]*callback

// Server represents a RPC server
type Server struct {
	listener   net.Listener
	serviceMap serviceRegistry
	mu         sync.Mutex      // protects codec collection
	codecs     codecCollection // active connections/codecs
}

// NewServer will create a new server instance with no registered handlers.
func NewServer() *Server {
	return &Server{
		serviceMap: make(serviceRegistry),
		codecs:     make(codecCollection),
	}
}

// Register publishes suitable methods of rcvr. Methods will be published
// with a dot between the rcrv and method, e.g. ChainManager.GetBlock.
func (s *Server) Register(rcvr interface{}) error {
	return s.register(rcvr, "", false)
}

// Register publishes suitable methods of rcvr. Methods will be published
// with a dot between the given name and method, e.g. ChainManager.GetBlock.
func (s *Server) RegisterName(name string, rcvr interface{}) error {
	return s.register(rcvr, name, true)
}

func (s *Server) register(rcvr interface{}, name string, useName bool) error {
	if s.serviceMap == nil {
		s.serviceMap = make(map[string]*service)
	}

	svc := new(service)
	svc.typ = reflect.TypeOf(rcvr)
	svc.rcvr = reflect.ValueOf(rcvr)

	sname := reflect.Indirect(svc.rcvr).Type().Name()
	if useName {
		sname = name
	}
	if sname == "" {
		return fmt.Errorf("no service name for type %s", svc.typ.String())
	}
	if !isExported(sname) && !useName {
		return fmt.Errorf("%s is not exported", sname)
	}

	if _, present := s.serviceMap[sname]; present {
		return fmt.Errorf("%s already registered", sname)
	}

	svc.name = sname
	svc.callbacks = suitableCallbacks(svc.typ)

	if len(svc.callbacks) == 0 {
		return fmt.Errorf("Service doesn't have any suitable methods to expose")
	}

	s.serviceMap[svc.name] = svc

	return nil
}

// Serve accepts connections and starts processing incoming requests.
func (s *Server) Serve(l net.Listener) {
	s.listener = l
	for {
		c, err := s.listener.Accept()
		if err != nil {
			glog.V(logger.Debug).Infof("%v\n", err)
			break
		}

		glog.V(logger.Debug).Infof("Accepted connection from %s\n", c.RemoteAddr())
		codec := NewECPCodec(c, c)
		s.mu.Lock()
		s.codecs[codec] = nil
		s.mu.Unlock()
		go s.serveCodec(codec)
	}

	// stopped accepting new connections, close existing
	for c, _ := range s.codecs {
		s.close(c)
	}
}

// Stop will stop listening for new connection and closes existing connections
func (s *Server) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// serveCodec start the server and uses the supplied codec to read and write
// requests and response. It will call Close on the codec when it returns.
func (s *Server) serveCodec(codec ServerCodec) error {
	defer s.close(codec)

	for {
		svc, c, argv, err := s.readRequest(codec)
		if err != nil {
			err2 := codec.WriteError(err)
			if !isRecoverable(err) {
				glog.V(logger.Error).Infoln(err)
				return err
			}
			if !isRecoverable(err2) {
				glog.V(logger.Error).Infoln(err2)
				return err2
			}
			continue
		}

		if res, err := s.call(c, svc.rcvr, argv); err != nil {
			if err = codec.WriteError(err); err != nil {
				if !isRecoverable(err) {
					glog.V(logger.Error).Infoln(err)
					return err
				}
			}
		} else {
			if res == nil { // callback has no return values
				if err = codec.WriteResponse(nil); err != nil {
					if !isRecoverable(err) {
						glog.V(logger.Error).Infoln(err)
						return err
					}
				}
			} else {
				values := make([]interface{}, len(res))
				for i := 0; i < len(res); i++ {
					values[i] = res[i].Interface()
				}

				if err = codec.WriteResponse(values); err != nil {
					if !isRecoverable(err) {
						glog.V(logger.Error).Infoln(err)
						return err
					}
				}
			}
		}
	}
}

func (s *Server) close(codec ServerCodec) {
	codec.Close()
	s.mu.Lock()
	delete(s.codecs, codec)
	s.mu.Unlock()
}

func (s *Server) call(c *callback, rcvr reflect.Value, argv []reflect.Value) ([]reflect.Value, error) {
	args := make([]reflect.Value, 1+len(argv))
	args[0] = rcvr
	for i := 0; i < len(argv); i++ {
		args[i+1] = argv[i]
	}
	rets := c.method.Func.Call(args)

	// check for error
	if c.errPos >= 0 && !rets[c.errPos].IsNil() {
		return nil, &callbackError{fmt.Sprintf("%s", rets[c.errPos])}
	}

	// don't send error when there is none
	if c.errPos > 0 {
		return rets[:c.errPos], nil
	}

	return rets, nil
}

func (s *Server) readRequest(codec ServerCodec) (svc *service, c *callback, argv []reflect.Value, err error) {
	req, err := codec.Read()
	if err != nil {
		return nil, nil, nil, err
	}

	// determine callback
	if svc = s.serviceMap[req.service]; svc == nil {
		return nil, nil, nil, &unknownServiceError{req.service}
	}

	if c = svc.callbacks[req.method]; c == nil {
		return nil, nil, nil, &unknownMethodError{req.service, req.method}
	}

	// decode params
	nArgs := len(c.args)
	if nArgs != len(req.args) {
		return nil, nil, nil, &invalidNumberOfArgumentsError{req.service, req.method, nArgs, len(req.args)}
	}

	argv = make([]reflect.Value, nArgs)
	for i := 0; i < nArgs; i++ {
		argv[i], err = convert(i, req.args[i], c.args[i])
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return
}
