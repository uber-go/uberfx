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

package fx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestWithAnnotated(t *testing.T) {
	type a struct {
		name string
	}

	newA := func() *a {
		return &a{name: "item a"}
	}

	newB := func() *a {
		return &a{name: "item b"}
	}

	newC := func() *a {
		return &a{name: "item c"}
	}

	t.Run("Provided", func(t *testing.T) {
		var inA *a
		var inB *a
		var inC *a
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name:   "a",
					Target: newA,
				},
				fx.Annotated{
					Group:  "b",
					Target: newB,
				},
				fx.Annotated{
					Name:   "c",
					Target: newC,
				},
			),
			fx.Invoke(fx.WithAnnotated(fx.NameAnnotation("a"), fx.GroupAnnotation("b"), fx.NameAnnotation("c"))(func(aa *a, bb []*a, cc *a) error {
				inA = aa
				inB = bb[0]
				inC = cc
				return nil
			})),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, inA, "expected a to be injected")
		assert.NotNil(t, inB, "expected b to be injected")
		assert.NotNil(t, inC, "expected c to be injected")
		assert.Equal(t, "item a", inA.name, "expected to get name 'item a'")
		assert.Equal(t, "item b", inB.name, "expected to get name 'item b'")
		assert.Equal(t, "item c", inC.name, "expected to get name 'item c'")
	})
}

func TestWithAnnotatedError(t *testing.T) {
	type a struct {
		name string
	}

	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("Provided", func(t *testing.T) {
		app := NewForTest(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
			),
			fx.Invoke(fx.WithAnnotated(fx.NameAnnotation("foo"))("")),
		)
		err := app.Err()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "WithAnnotated returned function must be called with a function")
	})
}
