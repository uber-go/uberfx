// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package fxyarpc

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/pkg/errors"

	"go.uber.org/fx"
	"go.uber.org/fx/config"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/transport/http"
	tch "go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
)

var (
	_dispatcherMu sync.Mutex
	// Function to create a dispatcher
	_dispatcherFn dispatcherFn = defaultYARPCDispatcher
	// Function to start a dispatcher
	_starterFn starterFn = defaultYARPCStarter
)

type dispatcherFn func(m *Module, l *zap.Logger, yc *yarpc.Config) fx.Component
type starterFn func(l *zap.Logger) fx.Component

func defaultYARPCDispatcher(m *Module, l *zap.Logger, yc *yarpc.Config) fx.Component {
	return func() (*yarpc.Dispatcher, error) {
		m.d = yarpc.NewDispatcher(*yc)
		return m.d, nil
	}
}

func defaultYARPCStarter(l *zap.Logger) fx.Component {
	return func(d *yarpc.Dispatcher, t *Transports) (*fake, error) {
		d.Register(t.Ts)

		l.Info("Starting the dispatcher")
		if err := d.Start(); err != nil {
			l.Error("Error starting the dispatcher", zap.Error(err))
			return nil, err
		}
		l.Info("Dispatcher started")
		return &fake{}, nil
	}
}

// RegisterDispatcher allows you to override the YARPC dispatcher registration
// TODO: Alternate solution to Register functions
func RegisterDispatcher(dispatchFn dispatcherFn) {
	_dispatcherMu.Lock()
	defer _dispatcherMu.Unlock()
	_dispatcherFn = dispatchFn
}

// StarterFn overrides start for dispatcher, e.g. attach some metrics with start.
type StarterFn func(dispatcher *yarpc.Dispatcher) error

// RegisterStarter allows you to override function that starts a dispatcher.
func RegisterStarter(startFn starterFn) {
	_dispatcherMu.Lock()
	defer _dispatcherMu.Unlock()
	_starterFn = startFn
}

// ServiceCreateFunc creates a YARPC service from a service host
type ServiceCreateFunc func(...interface{}) ([]transport.Procedure, error)

// Module foo
type Module struct {
	l  *zap.Logger
	fn fx.Component
	d  *yarpc.Dispatcher
}

// New foo
func New(fn fx.Component) *Module {
	// TODO: Check fn types
	return &Module{fn: fn}
}

// Name foo
func (m *Module) Name() string {
	return "yarpc"
}

// Transports foo
type Transports struct {
	Ts []transport.Procedure
}

type fake struct{}

// Constructor foo
func (m *Module) Constructor(core fx.Core) []fx.Component {
	// TODO: once #Constructors => []Component refactor is complete
	// this function needs to be split into two.
	// The first one would require config and create a dispatcher
	// The second one would require dispatcher and transports and:
	//		- Register transports in the dispatcher
	//	  - Start the dispatcher
	m.l = core.Logger().With(zap.String("module", "yarpc"))
	var c yarpcConfig
	// TODO: yarpc -> modules.yarpc
	if err := core.Config().Get("yarpc").Populate(&c); err != nil {
		panic(err)
	}
	svcname := core.Config().Get(config.ServiceNameKey).AsString()
	inb, err := prepareInbounds(c.Inbounds, svcname)
	if err != nil {
		panic(err)
	}
	yc := yarpc.Config{
		Name:     svcname,
		Inbounds: inb,
	}
	return []fx.Component{
		m.fn,
		_dispatcherFn(m, m.l, &yc),
		_starterFn(m.l),
	}
}

// Stop the dispatcher
func (m *Module) Stop() {
	if m.d != nil {
		m.l.Info("Stopping the dispatcher")
		if err := m.d.Stop(); err != nil {
			panic("Failed to stop dispatcher...")
		}
	}
}

type yarpcConfig struct {
	transports transports
	Inbounds   []Inbound
}

type transports struct {
	inbounds []transport.Inbound
}

// Inbound is a union that configures how to configure a single inbound.
type Inbound struct {
	TChannel *Address
	HTTP     *Address
}

// Address is a struct that have a required port for tchannel/http transports.
// TODO(alsam) make it optional
type Address struct {
	Port int
}

func (i *Inbound) String() string {
	if i == nil {
		return ""
	}
	http := "none"
	if i.HTTP != nil {
		http = strconv.Itoa(i.HTTP.Port)
	}
	tchannel := "none"
	if i.TChannel != nil {
		tchannel = strconv.Itoa(i.TChannel.Port)
	}
	return fmt.Sprintf("Inbound:{HTTP: %s; TChannel: %s}", http, tchannel)
}

// Iterate over all inbounds and prepare corresponding transports
func prepareInbounds(inbounds []Inbound, serviceName string) (transportsIn []transport.Inbound, err error) {
	transportsIn = make([]transport.Inbound, 0, 2*len(inbounds))
	for _, in := range inbounds {
		if h := in.HTTP; h != nil {
			transportsIn = append(
				transportsIn,
				http.NewTransport().NewInbound(fmt.Sprintf(":%d", h.Port)))
		}

		if t := in.TChannel; t != nil {
			chn, err := tch.NewChannelTransport(
				tch.ServiceName(serviceName),
				tch.ListenAddr(fmt.Sprintf(":%d", t.Port)))

			if err != nil {
				return nil, errors.Wrap(err, "can't create tchannel transport")
			}

			transportsIn = append(transportsIn, chn.NewInbound())
		}
	}

	return transportsIn, nil
}
