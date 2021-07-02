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
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnStartSuccess(t *testing.T) {
	type A string
	type B string
	type ctxKey struct{}

	type Params struct {
		In

		A A
		B B
	}

	var called []string

	ctxWithVal := func(val string) context.Context {
		return context.WithValue(context.Background(), ctxKey{}, val)
	}
	ctxVal := func(ctx context.Context) interface{} {
		return ctx.Value(ctxKey{})
	}

	// No arguments
	noArgs := func() {
		called = append(called, "noArgs")
	}
	hasArgs := func(a A, b B) {
		called = append(called, fmt.Sprintf("hasArgs(%v,%v)", a, b))
	}
	hasCtxAndArgs := func(ctx context.Context, a A) {
		called = append(called, fmt.Sprintf("hasCtxAndArgs(%v,%v)", ctx != nil, a))
	}
	hasParams := func(p Params) {
		called = append(called, fmt.Sprintf("hasParams(%v,%v)", p.A, p.B))
	}
	hasStopper := func(a A) Stopper {
		called = append(called, fmt.Sprintf("hasStopper(%v)", a))
		return func(ctx context.Context) error {
			called = append(called, fmt.Sprintf("hasStopper-stopper(%v)", a))
			return nil
		}
	}
	hasStopperErr := func(ctx context.Context, b B) (Stopper, error) {
		called = append(called, fmt.Sprintf("hasStopperErr(%v,%v)", ctxVal(ctx), b))
		return func(ctx context.Context) error {
			called = append(called, fmt.Sprintf("hasStopperErr-stopper(%v,%v)", ctxVal(ctx), b))
			return nil
		}, nil
	}

	app := New(
		Provide(
			func() A { return "val-A" },
			func() B { return "val-B" },
		),

		OnStart(noArgs, hasArgs),
		OnStart(hasCtxAndArgs),
		OnStart(hasParams),
		OnStart(hasStopper),
		OnStart(hasStopperErr),
	)

	startCtx := ctxWithVal("start-Ctx")
	stopCtx := ctxWithVal("stop-Ctx")

	require.NoError(t, app.Err(), "Failed to create app")
	require.NoError(t, app.Start(startCtx))
	require.NoError(t, app.Stop(stopCtx))

	want := []string{
		"noArgs",
		"hasArgs(val-A,val-B)",
		"hasCtxAndArgs(true,val-A)",
		"hasParams(val-A,val-B)",
		"hasStopper(val-A)",
		"hasStopperErr(start-Ctx,val-B)",

		// Stops happen at the end in inverse order to start.
		"hasStopperErr-stopper(stop-Ctx,val-B)",
		"hasStopper-stopper(val-A)",
	}
	assert.Equal(t, want, called)
}

func TestOnStartError(t *testing.T) {
	t.Run("OnStart returns error", func(t *testing.T) {
		onStart := func() error {
			return errors.New("onStart failed")
		}
		app := New(
			OnStart(onStart),
		)

		err := app.Start(context.Background())
		require.Error(t, err, "Start should fail due to OnStart error")
		assert.Contains(t, err.Error(), "onStart failed")
	})

	t.Run("OnStop returns error", func(t *testing.T) {
		onStart := func() (Stopper, error) {
			return func(context.Context) error {
				return errors.New("onStop failed")
			}, nil
		}
		app := New(
			OnStart(onStart),
		)

		require.NoError(t, app.Start(context.Background()), "Start should not fail")

		err := app.Stop(context.Background())
		require.Error(t, err, "Start should fail due to OnStart error")
		assert.Contains(t, err.Error(), "onStop failed")
	})
}

func TestOnStartInvalidType(t *testing.T) {
	tests := []struct {
		msg     string
		onStart interface{}
		wantErr string
	}{
		{
			msg:     "onstart is not a function",
			onStart: "onStart",
			wantErr: "OnStart must be passed a function, got string",
		},
		{
			msg:     "ctx is not first arg and not in graph",
			onStart: func(s string, ctx context.Context) {},
			wantErr: "type context.Context is not in the container",
		},
		{
			msg:     "onStart returns single unsupported type only",
			onStart: func() string { return "s" },
			wantErr: "only return an error or a Stopper",
		},
		{
			msg:     "onStart returns unsupported type and error",
			onStart: func() (string, error) { return "s", nil },
			wantErr: "only return (Stopper, error)",
		},
		{
			msg:     "onStart returns 3 results",
			onStart: func() (Stopper, string, error) { return nil, "s", nil },
			wantErr: "more than 2 returns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			app := New(
				Provide(func() string { return "str" }),
				OnStart(tt.onStart),
			)
			err := app.Err()
			require.Error(t, err, "Should fail to create app")
			assert.Contains(t, err.Error(), tt.wantErr, "Unexpected error")
		})
	}
}
