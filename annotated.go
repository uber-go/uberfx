// Copyright (c) 2020-2021 Uber Technologies, Inc.
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
	"fmt"
	"reflect"
	"strings"

	"go.uber.org/fx/internal/fxreflect"
)

// Annotated annotates a constructor provided to Fx with additional options.
//
// For example,
//
//   func NewReadOnlyConnection(...) (*Connection, error)
//
//   fx.Provide(fx.Annotated{
//     Name: "ro",
//     Target: NewReadOnlyConnection,
//   })
//
// Is equivalent to,
//
//   type result struct {
//     fx.Out
//
//     Connection *Connection `name:"ro"`
//   }
//
//   fx.Provide(func(...) (result, error) {
//     conn, err := NewReadOnlyConnection(...)
//     return result{Connection: conn}, err
//   })
//
// Annotated cannot be used with constructors which produce fx.Out objects.
//
// When used with fx.Supply, the target is a value rather than a constructor function.
type Annotated struct {
	// If specified, this will be used as the name for all non-error values returned
	// by the constructor. For more information on named values, see the documentation
	// for the fx.Out type.
	//
	// A name option may not be provided if a group option is provided.
	Name string

	// If specified, this will be used as the group name for all non-error values returned
	// by the constructor. For more information on value groups, see the package documentation.
	//
	// A group option may not be provided if a name option is provided.
	//
	// Similar to group tags, the group name may be followed by a `,flatten`
	// option to indicate that each element in the slice returned by the
	// constructor should be injected into the value group individually.
	Group string

	// Target is the constructor or value being annotated with fx.Annotated.
	Target interface{}
}

func (a Annotated) String() string {
	var fields []string
	if len(a.Name) > 0 {
		fields = append(fields, fmt.Sprintf("Name: %q", a.Name))
	}
	if len(a.Group) > 0 {
		fields = append(fields, fmt.Sprintf("Group: %q", a.Group))
	}
	if a.Target != nil {
		fields = append(fields, fmt.Sprintf("Target: %v", fxreflect.FuncName(a.Target)))
	}
	return fmt.Sprintf("fx.Annotated{%v}", strings.Join(fields, ", "))
}

// Annotation can be passed to Annotate(f interface{}, anns ...Annotation)
// for annotating the parameter and result types of a function.
type Annotation interface {
	getAnnotatedType(reflect.Type) []reflect.Type
}

type paramTags struct {
	tags []string
}

func (p paramTags) getAnnotatedType(fType reflect.Type) []reflect.Type {
	annotatedParams := []reflect.StructField{{
		Name:      "In",
		Type:      reflect.TypeOf(In{}),
		Anonymous: true,
	}}

	for i := 0; i < fType.NumIn(); i++ {
		structField := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: fType.In(i),
		}
		if i < len(p.tags) {
			structField.Tag = reflect.StructTag(p.tags[i])
		}
		annotatedParams = append(annotatedParams, structField)
	}

	paramTypes := []reflect.Type{reflect.StructOf(annotatedParams)}
	return paramTypes
}

// ParamTags is an Annotation that annotates the parameter(s) of a function.
// When multiple tags are specified, each tag is mapped to the corresponding
// positional parameter.
func ParamTags(tags ...string) Annotation {
	p := paramTags{tags}
	return p
}

type resultTags struct {
	tags []string
}

func (resultTags) getAnnotatedType(fType reflect.Type) []reflect.Type {
	annotatedResult := []reflect.StructField{{
		Name:      "Out",
		Type:      reflect.TypeOf(Out{}),
		Anonymous: true,
	}}

	for i := 0; i < fType.NumOut(); i++ {
		structField := reflect.StructField{
			Name: fmt.Sprintf("Field%d", i),
			Type: fType.Out(i),
		}
		annotatedResult = append(annotatedResult, structField)
	}

	resultTypes := []reflect.Type{reflect.StructOf(annotatedResult)}
	return resultTypes
}

// ResultTags is an Annotation that annotates the result(s) of a function.
// When multiple tags are specified, each tag is mapped to the corresponding
// positional result.
func ResultTags(tags ...string) Annotation {
	r := resultTags{tags}
	return r
}

// Annotate lets you annotate a function's paramter and returns with tags
// without you having to declare separate struct definitions for them.
//
// For example,
//   func NewGateway(ro, rw *db.Conn) *Gateway { ... }
//   fx.Provide(
//     fx.Annotate(
//       NewGateway,
//       fx.ParamTags(`name:"ro" optional:"true"`, `name:"rw"`),
//       fx.ResultTags(`name:"foo"`),
//     ),
//   )
//
// Is equivalent to,
//
//  type params struct {
//    fx.In
//
//    RO *db.Conn `name:"ro" optional:"true"`
//    RW *db.Conn `name:"rw"`
//  }
//
//  type result struct {
//    fx.Out
//
//    GW *Gateway `name:"foo"`
//   }
//
//   fx.Provide(func(p params) result {
//     return result{GW: NewGateway(p.RO, p.RW)}
//   })
//
// If there are more Annotations are specified than than one per
// Annotation type, none of the Annotations will be applied.
//
// For example,
//
// fx.Provide(
//   fx.Annotate(
//     NewGateWay,
//     fx.ParamTags(`name:"ro" optional:"true"`),
//     fx.ParamTags(`name:"rw"),
//     fx.ResultTags(`name:"foo"`)
//   )
// )
//
// is considered an invalid usage and will not apply any of the
// Annotations to NewGateway.
//
// If more tags are given than the number of parameters/results, only
// the ones up to the number of parameters/results will be applied.
func Annotate(f interface{}, anns ...Annotation) interface{} {
	fVal := reflect.ValueOf(f)
	fType := fVal.Type()
	numIn := fType.NumIn()
	numOut := fType.NumOut()

	if !verifyAnnotation(numIn, numOut, anns...) {
		return f
	}

	var ins []reflect.Type
	var outs []reflect.Type

	annotatedIn := false
	annotatedOut := false

	for _, ann := range anns {
		if pTags, ok := ann.(paramTags); ok {
			ins = pTags.getAnnotatedType(fType)
			annotatedIn = true
		}
		if rTags, ok := ann.(resultTags); ok {
			outs = rTags.getAnnotatedType(fType)
			annotatedOut = true
		}
	}

	if !annotatedIn {
		ins = make([]reflect.Type, numIn)
		for i := 0; i < fType.NumIn(); i++ {
			ins[i] = fType.In(i)
		}
	}
	if !annotatedOut {
		outs = make([]reflect.Type, numOut)
		for i := 0; i < fType.NumOut(); i++ {
			outs[i] = fType.Out(i)
		}
	}

	var newF func([]reflect.Value) []reflect.Value

	if annotatedIn && annotatedOut {
		newF = func(args []reflect.Value) []reflect.Value {
			fParams := make([]reflect.Value, numIn)
			params := args[0]
			for i := 0; i < numIn; i++ {
				fParams[i] = params.Field(i + 1)
			}
			fResults := fVal.Call(fParams)
			results := reflect.New(outs[0]).Elem()
			for i := 0; i < numOut; i++ {
				results.FieldByName(fmt.Sprintf("Field%d", i)).Set(fResults[i])
			}
			return []reflect.Value{results}
		}
	} else if annotatedIn {
		newF = func(args []reflect.Value) []reflect.Value {
			fParams := make([]reflect.Value, numIn)
			params := args[0]
			for i := 0; i < numIn; i++ {
				fParams[i] = params.Field(i + 1)
			}
			return fVal.Call(fParams)
		}
	} else { // annotatedOut
		newF = func(args []reflect.Value) []reflect.Value {
			fResults := fVal.Call(args)
			// wrap the result in annotated struct
			results := reflect.New(outs[0]).Elem()
			for i := 0; i < numOut; i++ {
				results.FieldByName(fmt.Sprintf("Field%d", i)).Set(fResults[i])
			}
			return []reflect.Value{results}
		}
	}

	annotatedFuncType := reflect.FuncOf(ins, outs, false)
	annotatedFunc := reflect.MakeFunc(annotatedFuncType, newF)
	return annotatedFunc.Interface()
}

func verifyAnnotation(numIn int, numOut int, anns ...Annotation) bool {
	sawParamAnn := false
	sawResultAnn := false

	for _, ann := range anns {
		if _, ok := ann.(paramTags); ok {
			if sawParamAnn {
				return false
			}
			sawParamAnn = true
		}
		if _, ok := ann.(resultTags); ok {
			if sawResultAnn {
				return false
			}
			sawResultAnn = true
		}
	}
	return true
}
