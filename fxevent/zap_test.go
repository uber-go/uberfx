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

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

type testLogSpy struct {
	testing.TB
	Messages []string
}

func newTestLogSpy(tb testing.TB) *testLogSpy {
	return &testLogSpy{TB: tb}
}

func (t *testLogSpy) Logf(format string, args ...interface{}) {
	// Log messages are in the format,
	//
	//   2017-10-27T13:03:01.000-0700	DEBUG	your message here	{data here}
	//
	// We strip the first part of these messages because we can't really test
	// for the timestamp from these tests.
	m := fmt.Sprintf(format, args...)
	m = m[strings.IndexByte(m, '\t')+1:]
	t.Messages = append(t.Messages, m)
	t.TB.Log(m)
}

func (t *testLogSpy) AssertMessages(msgs ...string) {
	assert.Equal(t.TB, msgs, t.Messages, "logged messages did not match")
}

func (t *testLogSpy) Reset() {
	t.Messages = t.Messages[:0]
}

func TestZapLogger(t *testing.T) {
	t.Parallel()

	ts := newTestLogSpy(t)
	logger := zaptest.NewLogger(ts)
	zapLogger := ZapLogger{Logger: logger}

	t.Run("LifecycleOnStart", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&LifecycleOnStart{CallerName: "bytes.NewBuffer"})
		ts.AssertMessages("INFO\tstarting\t{\"caller\": \"bytes.NewBuffer\"}")
	})
	t.Run("LifecycleOnStop", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&LifecycleOnStop{CallerName: "bytes.NewBuffer"})
		ts.AssertMessages("INFO\tstopping\t{\"caller\": \"bytes.NewBuffer\"}")
	})
	t.Run("ApplyOptionsError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&ApplyOptionsError{Err: fmt.Errorf("some error")})
		ts.AssertMessages("ERROR\terror encountered while applying options\t{\"error\": \"some error\"}")
	})

	t.Run("Supply", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&Supply{Constructor: bytes.NewBuffer})
		ts.AssertMessages("INFO\tsupplying\t{\"constructor\": \"bytes.NewBuffer()\", \"type\": \"*bytes.Buffer\"}")
	})
	t.Run("Provide", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&Provide{bytes.NewBuffer})
		ts.AssertMessages("INFO\tproviding\t{\"constructor\": \"bytes.NewBuffer()\", \"type\": \"*bytes.Buffer\"}")
	})
	t.Run("Invoke", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&Invoke{bytes.NewBuffer})
		ts.AssertMessages("INFO\tinvoke\t{\"function\": \"bytes.NewBuffer()\"}")
	})
	t.Run("InvokeFailed", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&InvokeFailed{
			Function: bytes.NewBuffer,
			Err:      fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfx.Invoke failed\t{\"error\": \"some error\", \"stack\": \"\", \"function\": \"bytes.NewBuffer()\"}")
	})
	t.Run("StartFailureError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&StartFailureError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfailed to start\t{\"error\": \"some error\"}")
	})
	t.Run("StopSignal", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&StopSignal{
			Signal: os.Interrupt,
		})
		ts.AssertMessages("INFO\treceived signal\t{\"signal\": \"INTERRUPT\"}")
	})
	t.Run("StopError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&StopError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tfailed to stop cleanly\t{\"error\": \"some error\"}")
	})
	t.Run("StartRollbackError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&StartRollbackError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tcould not rollback cleanly\t{\"error\": \"some error\"}")
	})
	t.Run("StartError", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&StartError{
			Err: fmt.Errorf("some error"),
		})
		ts.AssertMessages("ERROR\tstartup failed, rolling back\t{\"error\": \"some error\"}")
	})
	t.Run("Running", func(t *testing.T) {
		defer ts.Reset()
		zapLogger.LogEvent(&Running{})
		ts.AssertMessages("INFO\trunning")
	})
}