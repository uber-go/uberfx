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
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConsoleLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give Event
		want string
	}{
		{
			name: "LifecycleHookStart",
			give: &LifecycleHookStart{CallerName: "bytes.NewBuffer"},
			want: "[Fx] START		bytes.NewBuffer\n",
		},
		{
			name: "LifecycleHookStop",
			give: &LifecycleHookStop{CallerName: "bytes.NewBuffer"},
			want: "[Fx] STOP		bytes.NewBuffer\n",
		},
		{
			name: "ProvideError",
			give: &Provide{Err: errors.New("some error")},
			want: "[Fx] Error after options were applied: some error\n",
		},
		{
			name: "Supplied",
			give: &Supplied{TypeName: "*bytes.Buffer"},
			want: "[Fx] SUPPLY	*bytes.Buffer\n",
		},
		{
			name: "Provide",
			give: &Provide{bytes.NewBuffer, []string{"*bytes.Buffer"}, nil},
			want: "[Fx] PROVIDE	*bytes.Buffer <= bytes.NewBuffer()\n",
		},
		{
			name: "Invoke",
			give: &Invoke{Function: bytes.NewBuffer, Err: nil},
			want: "[Fx] INVOKE		bytes.NewBuffer()\n",
		},
		{
			name: "InvokeError",
			give: &Invoke{
				Function:   bytes.NewBuffer,
				Err:        errors.New("some error"),
				Stacktrace: "foo()\n\tbar/baz.go:42\n",
			},
			want: joinLines(
				"[Fx] fx.Invoke(bytes.NewBuffer()) called from:",
				"foo()",
				"	bar/baz.go:42",
				"Failed: some error",
			),
		},
		{
			name: "StartError",
			give: &Started{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to start: some error\n",
		},
		{
			name: "Stop",
			give: &Stop{Signal: os.Interrupt},
			want: "[Fx] INTERRUPT\n",
		},
		{
			name: "StopError",
			give: &Stop{Err: errors.New("some error")},
			want: "[Fx] ERROR		Failed to stop cleanly: some error\n",
		},
		{
			name: "RollbackError",
			give: &Rollback{Err: errors.New("some error")},
			want: "[Fx] ERROR		Couldn't roll back cleanly: some error\n",
		},
		{
			name: "Rollback",
			give: &Rollback{StartErr: errors.New("some error")},
			want: "[Fx] ERROR		Start failed, rolling back: some error\n",
		},
		{
			name: "Started",
			give: &Started{},
			want: "[Fx] RUNNING\n",
		},
		{
			name: "CustomLoggerError",
			give: &CustomLogger{Err: errors.New("great sadness")},
			want: "[Fx] ERROR		Failed to construct custom logger: great sadness\n",
		},
		{
			name: "CustomLogger",
			give: &CustomLogger{
				Function: func() Logger { panic("should not run") },
			},
			want: "[Fx] LOGGER	Setting up custom logger from go.uber.org/fx/fxevent.TestConsoleLogger.func1()\n",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			(&ConsoleLogger{W: &buff}).LogEvent(tt.give)

			assert.Equal(t, tt.want, buff.String())
		})
	}
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
