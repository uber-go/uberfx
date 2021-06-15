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

package fxtest

import (
	"context"
	"fmt"
	"io"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/internal/fxlog"
	"go.uber.org/fx/internal/lifecycle"
	"go.uber.org/fx/internal/testutil"
	"go.uber.org/zap/zapcore"
)

// If a testing.T is unspecified, degarde to printing to stderr to provide
// meaningful messages.
type panicT struct {
	W io.Writer // stream to which we'll write messages

	// lastError message written to the stream with Errorf. We'll use this
	// as the panic message if FailNow is called.
	lastErr string
}

var _ TB = &panicT{}

func (t *panicT) format(s string, args ...interface{}) string {
	return fmt.Sprintf(s, args...)
}

func (t *panicT) Logf(s string, args ...interface{}) {
	fmt.Fprintln(t.W, t.format(s, args...))
}

func (t *panicT) Errorf(s string, args ...interface{}) {
	t.lastErr = t.format(s, args...)
	fmt.Fprintln(t.W, t.lastErr)
}

func (t *panicT) FailNow() {
	if len(t.lastErr) > 0 {
		panic(t.lastErr)
	}

	panic("test lifecycle failed")
}

// Helper used by lifecycle.Start and lifecycle.Stop to drain out
// caller/HookRecord channels that should be ignored
func runWithNoopChan(
	ctx context.Context,
	f func(context.Context, chan string, chan lifecycle.HookRecord) error) error {
	c := make(chan error, 1)
	callerChan := make(chan string, 1)
	recordChan := make(chan lifecycle.HookRecord, 1)

	go func() { c <- f(ctx, callerChan, recordChan) }()
	for {
		select {
		case err := <-c:
			return err
		// Ignore caller/hookrecord channel
		case <-callerChan:
			continue
		case <-recordChan:
			continue
		}
	}
}

// Lifecycle is a testing spy for fx.Lifecycle. It exposes Start and Stop
// methods (and some test-specific helpers) so that unit tests can exercise
// hooks.
type Lifecycle struct {
	t  TB
	lc *lifecycle.Lifecycle
}

var _ fx.Lifecycle = (*Lifecycle)(nil)

// NewLifecycle creates a new test lifecycle.
func NewLifecycle(t TB) *Lifecycle {
	var ws zapcore.WriteSyncer
	if t != nil {
		ws = testutil.WriteSyncer{T: t}
	} else {
		// Retain the old behavior of printing to stderr if a testing.T
		// is not provided.
		ws = zapcore.AddSync(os.Stderr)
		t = &panicT{W: os.Stderr}
	}
	return &Lifecycle{
		lc: lifecycle.New(fxlog.DefaultLogger(ws)),
		t:  t,
	}
}

// Start executes all registered OnStart hooks in order, halting at the first
// hook that doesn't succeed.
func (l *Lifecycle) Start(ctx context.Context) error {
	return runWithNoopChan(ctx, l.lc.Start)
}

// RequireStart calls Start with context.Background(), failing the test if an
// error is encountered.
func (l *Lifecycle) RequireStart() *Lifecycle {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := l.Start(ctx); err != nil {
		l.t.Errorf("lifecycle didn't start cleanly: %v", err)
		l.t.FailNow()
	}
	return l
}

// Stop calls all OnStop hooks whose OnStart counterpart was called, running
// in reverse order.
//
// If any hook returns an error, execution continues for a best-effort
// cleanup. Any errors encountered are collected into a single error and
// returned.
func (l *Lifecycle) Stop(ctx context.Context) error {
	return runWithNoopChan(ctx, l.lc.Stop)
}

// RequireStop calls Stop with context.Background(), failing the test if an error
// is encountered.
func (l *Lifecycle) RequireStop() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := l.Stop(ctx); err != nil {
		l.t.Errorf("lifecycle didn't stop cleanly: %v", err)
		l.t.FailNow()
	}
}

// Append registers a new Hook.
func (l *Lifecycle) Append(h fx.Hook) {
	l.lc.Append(lifecycle.Hook{
		OnStart: h.OnStart,
		OnStop:  h.OnStop,
	})
}
