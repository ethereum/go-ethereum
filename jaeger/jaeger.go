// Copyright 2021 The go-ethereum Authors
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

package jaeger

import (
	"errors"
	"io"

	"github.com/ethereum/go-ethereum/node"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"
)

type Jaeger struct {
	tracer opentracing.Tracer
	closer io.Closer
}

func New(node *node.Node) (*Jaeger, error) {
	cfg := &config.Configuration{
		ServiceName: "main",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.NewTracer(config.Logger(jaeger.StdLogger))
	if err != nil {
		return nil, err
	}

	j := &Jaeger{tracer, closer}
	node.RegisterLifecycle(j)
	return j, nil
}

func (j *Jaeger) Start() error {
	if opentracing.IsGlobalTracerRegistered() {
		return errors.New("Global tracer already registered. Jaeger being initialized twice.")
	}
	opentracing.SetGlobalTracer(j.tracer)
	return nil
}

func (j *Jaeger) Stop() error {
	return j.closer.Close()
}

func (j *Jaeger) Tracer() opentracing.Tracer {
	return j.tracer
}
