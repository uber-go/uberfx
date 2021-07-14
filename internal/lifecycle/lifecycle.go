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

package lifecycle

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/multierr"
)

// A Hook is a pair of start and stop callbacks, either of which can be nil,
// plus a string identifying the supplier of the hook.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error

	callerFrame fxreflect.Frame
}

// Lifecycle coordinates application lifecycle hooks.
type Lifecycle struct {
	logger       fxevent.Logger
	hooks        []Hook
	numStarted   int
	startRecords HookRecords
	stopRecords  HookRecords
	runningHook  Hook
	mu           sync.Mutex
}

// New constructs a new Lifecycle.
func New(logger fxevent.Logger) *Lifecycle {
	return &Lifecycle{logger: logger}
}

// Append adds a Hook to the lifecycle.
func (l *Lifecycle) Append(hook Hook) {
	// Save the caller's stack frame to report file/line number.
	if f := fxreflect.CallerStack(2, 0); len(f) > 0 {
		hook.callerFrame = f[0]
	}
	l.hooks = append(l.hooks, hook)
}

// Start runs all OnStart hooks, returning immediately if it encounters an
// error.
func (l *Lifecycle) Start(ctx context.Context) error {
	l.startRecords = make(HookRecords, 0, len(l.hooks))
	for _, hook := range l.hooks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if hook.OnStart != nil {
				l.logger.LogEvent(&fxevent.LifecycleHookStart{CallerName: hook.callerFrame.Function})

				l.mu.Lock()
				l.runningHook = hook
				l.mu.Unlock()

				begin := time.Now()
				if err := hook.OnStart(ctx); err != nil {
					return err
				}
				l.mu.Lock()
				l.startRecords = append(l.startRecords, HookRecord{
					CallerFrame: hook.callerFrame,
					Func:        hook.OnStart,
					Runtime:     time.Since(begin),
				})
				l.mu.Unlock()
			}
			l.numStarted++
		}
	}

	return nil
}

// Stop runs any OnStop hooks whose OnStart counterpart succeeded. OnStop
// hooks run in reverse order.
func (l *Lifecycle) Stop(ctx context.Context) error {
	var errs []error
	l.mu.Lock()
	l.stopRecords = make(HookRecords, 0, l.numStarted)
	l.mu.Unlock()

	// Run backward from last successful OnStart.
	for ; l.numStarted > 0; l.numStarted-- {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			hook := l.hooks[l.numStarted-1]
			if hook.OnStop == nil {
				continue
			}

			l.logger.LogEvent(&fxevent.LifecycleHookStop{CallerName: hook.callerFrame.Function})

			l.mu.Lock()
			l.runningHook = hook
			l.mu.Unlock()

			begin := time.Now()
			if err := hook.OnStop(ctx); err != nil {
				// For best-effort cleanup, keep going after errors.
				errs = append(errs, err)
			}
			l.mu.Lock()
			l.stopRecords = append(l.stopRecords, HookRecord{
				CallerFrame: hook.callerFrame,
				Func:        hook.OnStop,
				Runtime:     time.Since(begin),
			})
			l.mu.Unlock()
		}
	}

	return multierr.Combine(errs...)
}

// StartHookRecords returns the info of OnStart hooks that successfully ran till the end,
// including their caller and runtime. Used to report timeout errors on Start.
func (l *Lifecycle) StartHookRecords() HookRecords {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := make(HookRecords, len(l.startRecords))
	copy(r, l.startRecords)
	return r
}

// StopHookRecords returns the info of OnStop hooks that successfully ran till the end,
// including their caller and runtime. Used to report timeout errors on Stop.
func (l *Lifecycle) StopHookRecords() HookRecords {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := make(HookRecords, len(l.stopRecords))
	copy(r, l.stopRecords)
	return r
}

// RunningHookCaller returns the name of the hook that was running when a Start/Stop
// hook timed out.
func (l *Lifecycle) RunningHookCaller() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.runningHook.callerFrame.Function
}

// HookRecord keeps track of each Hook's execution time, the caller that appended the Hook, and function that ran as the Hook.
type HookRecord struct {
	CallerFrame fxreflect.Frame             // stack frame of the caller
	Func        func(context.Context) error // function that ran as sanitized name
	Runtime     time.Duration               // how long the hook ran
}

// HookRecords is a Stringer wrapper of HookRecord slice.
type HookRecords []HookRecord

func (rs HookRecords) Len() int {
	return len(rs)
}

func (rs HookRecords) Less(i, j int) bool {
	// Sort by runtime, greater ones at top.
	return rs[i].Runtime > rs[j].Runtime
}

func (rs HookRecords) Swap(i, j int) {
	rs[i], rs[j] = rs[j], rs[i]
}

// Used for logging startup errors.
func (rs HookRecords) String() string {
	var b strings.Builder
	for _, r := range rs {
		fmt.Fprintf(&b, "%s took %v from %s",
			fxreflect.FuncName(r.Func), r.Runtime, r.CallerFrame)
	}
	return b.String()
}

// Format implements fmt.Formatter to handle "%+v".
func (rs HookRecords) Format(w fmt.State, c rune) {
	if !w.Flag('+') {
		// Without %+v, fall back to String().
		io.WriteString(w, rs.String())
		return
	}

	for _, r := range rs {
		fmt.Fprintf(w, "\n%s took %v from:\n\t%+v",
			fxreflect.FuncName(r.Func),
			r.Runtime,
			r.CallerFrame)
	}
	fmt.Fprintf(w, "\n")
}
