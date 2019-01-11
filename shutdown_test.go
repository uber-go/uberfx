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
	"context"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestShutdown(t *testing.T) {
	t.Run("BroadcastsToMultipleChannels", func(t *testing.T) {
		var s fx.Shutdowner
		lv := newLifecycleVerifier(t)
		app := fxtest.New(
			t,
			lv,
			fx.Populate(&s),
		)

		done1, done2 := app.Done(), app.Done()
		app.RequireStart()
		assert.NoError(t, s.Shutdown(), "error in app shutdown")

		assert.Equal(t, syscall.SIGTERM, <-done1, "done channel 1 did not receive signal")
		assert.Equal(t, syscall.SIGTERM, <-done2, "done channel 2 did not receive signal")

		app.RequireStop()
		lv.verifyStartAndStopHooks()
	})

	t.Run("ErrorOnUnsentSignal", func(t *testing.T) {
		var s fx.Shutdowner
		lv := newLifecycleVerifier(t)
		app := fxtest.New(
			t,
			lv,
			fx.Populate(&s),
		)

		done := app.Done()
		assert.NoError(t, s.Shutdown(), "error returned from first shutdown call")
		app.RequireStart()

		assert.Error(t, s.Shutdown(), "no error returned when shutdown is called with a blocked channel")
		assert.Equal(t, syscall.SIGTERM, <-done, "done channel did not receive signal")

		app.RequireStop()
		lv.verifyStartAndStopHooks()
	})
}

type lifecycleVerifier struct {
	fx.Option

	started, stopped bool

	tb testing.TB
}

func newLifecycleVerifier(tb testing.TB) *lifecycleVerifier {
	lv := &lifecycleVerifier{tb: tb}
	lv.Option = fx.Invoke(func(l fx.Lifecycle) {
		l.Append(fx.Hook{
			OnStart: func(context.Context) error {
				lv.started = true
				return nil
			},
			OnStop: func(context.Context) error {
				lv.stopped = true
				return nil
			},
		})
	})

	return lv
}

func (lv *lifecycleVerifier) verifyStartAndStopHooks() {
	if lv.started != true {
		lv.tb.Errorf("app OnStart hooks were never triggered")
	}
	if lv.stopped != true {
		lv.tb.Errorf("app OnStop hooks were never triggered")
	}
}
