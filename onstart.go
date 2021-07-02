// Copyright (c) 2019 Uber Technologies, Inc.
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

package fx

import (
	"context"
	"fmt"
	"reflect"
)

var (
	_ctxType       = typeOfInterface(new(context.Context))
	_lifecycleType = typeOfInterface(new(Lifecycle))
	_errType       = typeOfInterface(new(error))
	_stopperType   = reflect.TypeOf(Stopper(nil))
)

// Stopper functions are are executed when the applicaton is stopped.
// They can be returned from a function passed in to OnStart.
type Stopper func(context.Context) error

type onStartType struct {
	hasCtx     bool
	hasErr     bool
	hasStopper bool
}

func typeOfInterface(newIfacePtr interface{}) reflect.Type {
	return reflect.TypeOf(newIfacePtr).Elem()
}

// OnStart registers functions that are executed when the application is started.
// Arguments for these invocations are built using the constructors provided by
// Provide. Passing multiple OnStart options appends new functions to the applications'
// existing list.
//
// Similar to Invokes, functions are always executed, and they are run in order.
// OnStart functions may return no values, or an error. Any other return values
// cause errors.
//
// If the first argument of the passed in function is a context.Context, then the
// context from Start is passed in.
//
// If any of OnStart functions return a Stopper as their last non-error
// argument, the Stopper will be queued up with OnStop.
//
//  fx.New(
//    ...,
//    OnStart(func(server *http.Server) Stopper {
//      go server.ListenAndServe()
//      return server.Shutdown
//    }),
//  )
//
// The above is equivalent to,
//
//  fx.New(
//    ...,
//    OnStart(func(server *http.Server) {
//      go server.ListenAndServe()
//    }),
//    OnStop(func(ctx context.Context, server *http.Server) error {
//      return server.Shutdown(ctx)
//    }),
//  )
func OnStart(funcs ...interface{}) Option {
	var opts []Option
	for _, f := range funcs {
		opts = append(opts, onStart(reflect.ValueOf(f)))
	}
	return Options(opts...)
}

func validateOnStartFunc(ft reflect.Type) (onStartType, error) {
	if kind := ft.Kind(); kind != reflect.Func {
		return onStartType{}, fmt.Errorf("OnStart must be passed a function, got %v", kind)
	}

	onStartDesc := onStartType{
		hasCtx: ft.NumIn() > 0 && ft.In(0) == _ctxType,
	}

	switch ft.NumOut() {
	case 0:
		// No return value, nothing to validate.
	case 1:
		switch ft.Out(0) {
		case _errType:
			onStartDesc.hasErr = true
		case _stopperType:
			onStartDesc.hasStopper = true
		default:
			return onStartDesc, fmt.Errorf("OnStart functions can only return an error or a Stopper, found %v", ft.Out(0))
		}
	case 2:
		if ft.Out(0) != _stopperType || ft.Out(1) != _errType {
			return onStartDesc, fmt.Errorf("OnStart functions can only return (Stopper, error), found (%v, %v)", ft.Out(0), ft.Out(1))
		}
		onStartDesc.hasErr = true
		onStartDesc.hasStopper = true
	default:
		return onStartDesc, fmt.Errorf("OnStart functions must not have more than 2 returns")
	}

	return onStartDesc, nil
}

func onStart(f reflect.Value) Option {
	ft := f.Type()

	onStartDesc, err := validateOnStartFunc(ft)
	if err != nil {
		return invokeErr(err)
	}

	// The invoke function will take a lifecycle, followed by any arguments to f.
	invokeInTypes := []reflect.Type{_lifecycleType}
	for i := 0; i < ft.NumIn(); i++ {
		if i == 0 && onStartDesc.hasCtx {
			continue
		}

		invokeInTypes = append(invokeInTypes, ft.In(i))
	}

	invokeFuncType := reflect.FuncOf(invokeInTypes, []reflect.Type{_errType}, false /* variadic */)

	invokeFunc := reflect.MakeFunc(invokeFuncType, func(args []reflect.Value) []reflect.Value {
		// The first argument to invoke is a lifecycle, let's extract that.
		lifecycle := args[0].Interface().(Lifecycle)
		var onStop Stopper
		lifecycle.Append(Hook{
			OnStart: func(ctx context.Context) error {
				var startArgs []reflect.Value

				if onStartDesc.hasCtx {
					startArgs = append(startArgs, reflect.ValueOf(ctx))
				}

				startArgs = append(startArgs, args[1:]...)
				res := f.Call(startArgs)

				// If the OnStop returns an error, then it must be the last argument
				if onStartDesc.hasErr {
					if err := res[len(res)-1].Interface(); err != nil {
						return err.(error)
					}
				}

				// If the OnStop returns a Stopper, it must be the first argument.
				if onStartDesc.hasStopper {
					onStop = res[0].Interface().(Stopper)
				}

				return nil
			},
			OnStop: func(ctx context.Context) error {
				if onStop == nil {
					return nil
				}

				return onStop(ctx)
			},
		})

		return []reflect.Value{reflect.New(_errType).Elem()}
	})

	return Invoke(invokeFunc.Interface())
}
