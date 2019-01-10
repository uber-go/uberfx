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
	"fmt"
	"os"
	"syscall"
)

// Shutdowner provides a method that can manually trigger the shutdown of the
// application by sending a signal to all open Done channels. Shutdowner works
// on applications using Run as well as Start, Done, and Stop. The Shutdowner is
// provided to all Fx applications.
type Shutdowner interface {
	Shutdown(...ShutdownOption) error
}

// Shutdown broadcasts a signal to all of the application's Done channels
// and begins the Stop process.
func (s *shutdowner) Shutdown(opts ...ShutdownOption) error {
	return s.broadcastSignal(syscall.SIGTERM)
}

// ShutdownOption provides a way to configure properties of the shutdown
// process. Currently, no options have been implemented.
type ShutdownOption interface {
	apply(*shutdowner)
}

type shutdowner struct {
	broadcastSignal func(os.Signal) error
}

func (app *App) shutdowner() Shutdowner {
	return &shutdowner{
		broadcastSignal: app.signalBroadcaster(),
	}
}

func (app *App) signalBroadcaster() func(os.Signal) error {
	return func(signal os.Signal) error {
		app.mu.RLock()
		defer app.mu.RUnlock()

		var unsent int
		for i, done := range app.dones {
			select {
			case done <- signal:
			default:
				// shutdown called when done channel has already received a
				// termination signal that has not been cleared
				unsent++
				app.logger.Printf("done channel %d at capacity, did not receive signal %v", i, signal)
			}
		}

		if unsent != 0 {
			err := fmt.Errorf("failed to send %v signal to %v out of %v channels", signal, unsent, len(app.dones))
			return err
		}

		return nil
	}
}
