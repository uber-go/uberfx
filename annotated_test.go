// Copyright (c) 2019-2021 Uber Technologies, Inc.
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/fxtest"
)

func TestAnnotated(t *testing.T) {
	type a struct {
		name string
	}
	type in struct {
		fx.In

		A *a `name:"foo"`
	}
	newA := func() *a {
		return &a{name: "foo"}
	}
	t.Run("Provide", func(t *testing.T) {
		var in in
		app := fxtest.New(t,
			fx.Provide(
				fx.Annotated{
					Name:   "foo",
					Target: newA,
				},
			),
			fx.Populate(&in),
		)
		defer app.RequireStart().RequireStop()
		assert.NotNil(t, in.A, "expected in.A to be injected")
		assert.Equal(t, "foo", in.A.name, "expected to get a type 'a' of name 'foo'")
	})
}

func TestAnnotatedWrongUsage(t *testing.T) {
	type a struct {
		name string
	}
	type in struct {
		fx.In

		A *a `name:"foo"`
	}
	newA := func() *a {
		return &a{name: "foo"}
	}

	t.Run("In Constructor", func(t *testing.T) {
		var in in
		app := fx.New(
			fx.WithLogger(func() fxevent.Logger {
				return fxtest.NewTestLogger(t)
			}),
			fx.Provide(
				func() fx.Annotated {
					return fx.Annotated{
						Name:   "foo",
						Target: newA,
					}
				},
			),
			fx.Populate(&in),
		)

		err := app.Err()
		require.Error(t, err)

		// Example:
		// fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor: fx.Provide received go.uber.org/fx_test.TestAnnotatedWrongUsage.func2.1() from:
		// go.uber.org/fx_test.TestAnnotatedWrongUsage.func2
		//         /.../fx/annotated_test.go:76
		// testing.tRunner
		//         /.../go/1.13.3/libexec/src/testing/testing.go:909
		assert.Contains(t, err.Error(), "fx.Annotated should be passed to fx.Provide directly, it should not be returned by the constructor")
		assert.Contains(t, err.Error(), "fx.Provide received go.uber.org/fx_test.TestAnnotatedWrongUsage")
		assert.Contains(t, err.Error(), "go.uber.org/fx_test.TestAnnotatedWrongUsage")
		assert.Contains(t, err.Error(), "/fx/annotated_test.go")
	})

	t.Run("Result Type", func(t *testing.T) {
		app := fx.New(
			fx.WithLogger(func() fxevent.Logger {
				return fxtest.NewTestLogger(t)
			}),
			fx.Provide(
				fx.Annotated{
					Name: "foo",
					Target: func() in {
						return in{A: &a{name: "foo"}}
					},
				},
			),
		)
		assert.Contains(t, app.Err().Error(), "embeds a dig.In", "expected error when result types were annotated")
	})
}

func TestAnnotatedString(t *testing.T) {
	tests := []struct {
		desc string
		give fx.Annotated
		want string
	}{
		{
			desc: "empty",
			give: fx.Annotated{},
			want: "fx.Annotated{}",
		},
		{
			desc: "name",
			give: fx.Annotated{Name: "foo"},
			want: `fx.Annotated{Name: "foo"}`,
		},
		{
			desc: "group",
			give: fx.Annotated{Group: "foo"},
			want: `fx.Annotated{Group: "foo"}`,
		},
		{
			desc: "name and group",
			give: fx.Annotated{Name: "foo", Group: "bar"},
			want: `fx.Annotated{Name: "foo", Group: "bar"}`,
		},
		{
			desc: "target",
			give: fx.Annotated{Target: func() {}},
			want: "fx.Annotated{Target: go.uber.org/fx_test.TestAnnotatedString.func1()}",
		},
		{
			desc: "name and target",
			give: fx.Annotated{Name: "foo", Target: func() {}},
			want: `fx.Annotated{Name: "foo", Target: go.uber.org/fx_test.TestAnnotatedString.func2()}`,
		},
		{
			desc: "group and target",
			give: fx.Annotated{Group: "foo", Target: func() {}},
			want: `fx.Annotated{Group: "foo", Target: go.uber.org/fx_test.TestAnnotatedString.func3()}`,
		},
		{
			desc: "name, group and target",
			give: fx.Annotated{Name: "foo", Group: "bar", Target: func() {}},
			want: `fx.Annotated{Name: "foo", Group: "bar", Target: go.uber.org/fx_test.TestAnnotatedString.func4()}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.give.String())
		})
	}
}
