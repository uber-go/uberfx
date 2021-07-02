// Copyright (c) 2020 Uber Technologies, Inc.
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
	"errors"
	"fmt"
	"reflect"
)

// Annotation this will be passed to WithAnnotated to identify which to be injected
type Annotation interface {
	isAnnotation()
}

type groupAnnotation struct {
	group string
}

func (groupAnnotation) isAnnotation() {}

// GroupAnnotation use group Annotated inject
func GroupAnnotation(group string) Annotation {
	return groupAnnotation{
		group: group,
	}
}

type nameAnnotation struct {
	name string
}

func (nameAnnotation) isAnnotation() {}

// NameAnnotation use name Annotated inject
func NameAnnotation(name string) Annotation {
	return nameAnnotation{
		name: name,
	}
}

// WithAnnotated allows to inject annotated options without declare your own struct
//
// For example,
//
//   func NewReadOnlyConnection(...) (*Connection, error)
//   fx.Provide(fx.Annotated{
//     Name: "ro",
//     Target: NewReadOnlyConnection,
//   })
//   fx.Supply(&Server{})
//
//   fx.Invoke(fx.WithAnnotated(fx.NameAnnotation("ro))(func (roConn *Connection, s *Server) error {
//     return nil
//   }))
//
// Is equivalent to,
//
//   type Params struct {
//     fx.In
//
//     Connection *Connection `name:"ro"`
//     Server *Server
//   }
//
//   fx.Invoke(func(params Params) error {
//      roConn := params.Connection
//      s := params.Server
//      return nil
//   })
//
// WithAnnotated takes an array of names, and returns function to be called with user function. names are in order.
func WithAnnotated(annos ...Annotation) func(interface{}) interface{} {
	numNames := len(annos)
	return func(f interface{}) interface{} {
		userFunc := reflect.ValueOf(f)
		userFuncType := userFunc.Type()
		if userFuncType.Kind() != reflect.Func {
			return func() error {
				return errors.New("WithAnnotated returned function must be called with a function")
			}
		}
		numArgs := userFuncType.NumIn()
		digInStructFields := []reflect.StructField{{
			Name:      "In",
			Anonymous: true,
			Type:      reflect.TypeOf(In{}),
		}}
		for i := 0; i < numArgs; i++ {
			name := fmt.Sprintf("Field%d", i)
			field := reflect.StructField{
				Name: name,
				Type: userFuncType.In(i),
			}
			if i < numNames { // namedArguments
				tag := ""
				annos[i].isAnnotation()
				switch anno := annos[i].(type) {
				case groupAnnotation:
					tag = fmt.Sprintf(`group:"%s"`, anno.group)
				case nameAnnotation:
					tag = fmt.Sprintf(`name:"%s"`, anno.name)
				}

				field.Tag = reflect.StructTag(tag)
			}
			digInStructFields = append(digInStructFields, field)
		}

		outs := make([]reflect.Type, userFuncType.NumOut())
		for i := 0; i < userFuncType.NumOut(); i++ {
			outs[i] = userFuncType.Out(i)
		}

		paramType := reflect.StructOf(digInStructFields)
		fxOptionFuncType := reflect.FuncOf([]reflect.Type{paramType}, outs, false)
		fxOptionFunc := reflect.MakeFunc(fxOptionFuncType, func(args []reflect.Value) []reflect.Value {
			callUserFuncINs := make([]reflect.Value, numArgs)
			params := args[0]
			for i := 0; i < numArgs; i++ {
				callUserFuncINs[i] = params.Field(i + 1)
			}
			return userFunc.Call(callUserFuncINs)
		})

		return fxOptionFunc.Interface()
	}
}
