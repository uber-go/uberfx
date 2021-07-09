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

package fxlog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxevent"
)

func TestSpy(t *testing.T) {
	var s Spy

	t.Run("empty spy", func(t *testing.T) {
		assert.Empty(t, s.Events(), "events must be empty")
		assert.Empty(t, s.EventTypes(), "event types must be empty")
	})

	s.LogEvent(&fxevent.Started{})
	t.Run("use after reset", func(t *testing.T) {
		assert.Equal(t, "Started", s.EventTypes()[0])
	})

	s.LogEvent(&fxevent.Provide{Err: fmt.Errorf("some error")})
	t.Run("some error", func(t *testing.T) {
		assert.Equal(t, "Provide", s.EventTypes()[1])
	})

	s.Reset()
	t.Run("reset", func(t *testing.T) {
		assert.Empty(t, s.Events(), "events must be empty")
		assert.Empty(t, s.EventTypes(), "event types must be empty")
	})

	s.LogEvent(&fxevent.Started{})
	t.Run("use after reset", func(t *testing.T) {
		assert.Equal(t, "Started", s.EventTypes()[0])
	})
}
