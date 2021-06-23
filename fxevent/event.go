// Copyright (c) 2021 Uber Technologies, Inc.
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

package fxevent

import "os"

// Event defines an event emitted by fx.
type Event interface {
	event() // Only fxlog can implement this interface.
}

// Passing events by type to make Event hashable in the future.
func (*LifecycleOnStart) event()   {}
func (*LifecycleOnStop) event()    {}
func (*ApplyOptionsError) event()  {}
func (*Supply) event()             {}
func (*Provide) event()            {}
func (*Invoke) event()             {}
func (*InvokeFailed) event()       {}
func (*StartFailureError) event()  {}
func (*StopSignal) event()         {}
func (*StopError) event()          {}
func (*StartError) event()         {}
func (*StartRollbackError) event() {}
func (*Running) event()            {}

// LifecycleOnStart is emitted for whenever an OnStart hook is executed
type LifecycleOnStart struct {
	CallerName string
}

// LifecycleOnStop is emitted for whenever an OnStart hook is executed
type LifecycleOnStop struct {
	CallerName string
}

// ApplyOptionsError is emitted whenever there is an error applying options.
type ApplyOptionsError struct {
	Err error
}

// Supply is emitted whenever a Provide was called with a constructor provided
// by fx.Supply.
type Supply struct {
	Constructor interface{}
}

// Provide is emitted whenever Provide was called and is not provided by fx.Supply.
type Provide struct {
	Constructor interface{}
}

// Invoke is emitted whenever a function is invoked.
type Invoke struct {
	Function interface{}
}

// InvokeFailed is emitted when fx.Invoke has failed.
type InvokeFailed struct {
	Function   interface{}
	Err        error
	Stacktrace string
}

// StartFailureError is emitted right before exiting after failing to start.
type StartFailureError struct{ Err error }

// StopSignal is emitted whenever application receives a signal after
// starting the application.
type StopSignal struct{ Signal os.Signal }

// StopError is emitted whenever we fail to stop cleanly.
type StopError struct{ Err error }

// StartError is emitted whenever a service fails to start.
type StartError struct{ Err error }

// StartRollbackError is emitted whenever we fail to rollback cleanly after
// a start error.
type StartRollbackError struct{ Err error }

// Running is emitted whenever an application is started successfully.
type Running struct{}