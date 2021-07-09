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

package fx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxevent"
	"go.uber.org/fx/internal/fxlog"
)

func TestLogBufferConnect(t *testing.T) {
	spy := new(fxlog.Spy)
	event := &fxevent.Started{}
	lb := &logBuffer{
		events: []fxevent.Event{event},
		logger: nil,
	}

	lb.Connect(spy)
	assert.Equal(t, spy.Events(), []fxevent.Event{event})
}

func TestLogBufferLog(t *testing.T) {
	spy := new(fxlog.Spy)
	event := &fxevent.Started{}
	lb := &logBuffer{
		events: nil,
		logger: nil,
	}

	lb.LogEvent(event)

	lb.Connect(spy)
	assert.Equal(t, spy.Events(), []fxevent.Event{event})
}
