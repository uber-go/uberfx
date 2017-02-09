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

package task

import (
	"sync"

	"github.com/pkg/errors"

	"golang.org/x/net/context"

	"go.uber.org/fx/modules"
	"go.uber.org/fx/service"
)

type globalBackend struct {
	backend Backend
	sync.RWMutex
}

var (
	_globalBackendMu sync.RWMutex
	_globalBackend   Backend = &NopBackend{}
	_asyncMod        service.Module
	_asyncModErr     error
	_once            sync.Once
)

// GlobalBackend returns global instance of the backend
// TODO (madhu): Make work with multiple backends
func GlobalBackend() Backend {
	_globalBackendMu.RLock()
	defer _globalBackendMu.RUnlock()
	return _globalBackend
}

// NewModule creates an async task queue module
func NewModule(createFunc BackendCreateFunc) service.ModuleCreateFunc {
	return func(mi service.ModuleCreateInfo) ([]service.Module, error) {
		mod, err := newAsyncModuleSingleton(mi, createFunc)
		return []service.Module{mod}, err
	}
}

func newAsyncModuleSingleton(
	mi service.ModuleCreateInfo, createFunc BackendCreateFunc,
) (service.Module, error) {
	_once.Do(func() {
		_asyncMod, _asyncModErr = newAsyncModule(mi, createFunc)
	})
	return _asyncMod, _asyncModErr
}

func newAsyncModule(
	mi service.ModuleCreateInfo, createFunc BackendCreateFunc,
) (service.Module, error) {
	backend, err := createFunc(mi.Host)
	if err != nil {
		return nil, err
	}
	_globalBackendMu.Lock()
	_globalBackend = backend
	_globalBackendMu.Unlock()
	// Register context implementation with the encoder so it can be used in async functions
	// TODO (madhu): Will be removed once moving context through the backend is figured out
	if err := _globalBackend.Encoder().Register(context.Background()); err != nil {
		return nil, errors.Wrap(err, "unable to register context with the backend encoder")
	}
	return &AsyncModule{
		Backend: backend,
		modBase: *modules.NewModuleBase("task", mi.Host, []string{}),
	}, nil
}

// BackendCreateFunc creates a backend implementation
type BackendCreateFunc func(host service.Host) (Backend, error)

// AsyncModule denotes the asynchronous task queue module
type AsyncModule struct {
	Backend
	modBase modules.ModuleBase
}
