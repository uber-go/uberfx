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
	"strings"

	"go.uber.org/fx/internal/fxreflect"
	"go.uber.org/zap"
)

// ZapLogger is an Fx event logger that logs events to Zap.
type ZapLogger struct {
	Logger *zap.Logger
}

var _ Logger = (*ZapLogger)(nil)

func (l *ZapLogger) LogEvent(event Event) {
	switch e := event.(type) {
	case *LifecycleOnStart:
		l.Logger.Info("starting", zap.String("caller", e.CallerName))
	case *LifecycleOnStop:
		l.Logger.Info("stopping", zap.String("caller", e.CallerName))
	case *ApplyOptionsError:
		l.Logger.Error("error encountered while applying options", zap.Error(e.Err))
	case *Supply:
		for _, rtype := range fxreflect.ReturnTypes(e.Constructor) {
			l.Logger.Info("supplying",
				zap.String("constructor", fxreflect.FuncName(e.Constructor)),
				zap.String("type", rtype),
			)
		}
	case *Provide:
		for _, rtype := range fxreflect.ReturnTypes(e.Constructor) {
			l.Logger.Info("providing",
				zap.String("constructor", fxreflect.FuncName(e.Constructor)),
				zap.String("type", rtype),
			)
		}
	case *Invoke:
		l.Logger.Info("invoke", zap.String("function", fxreflect.FuncName(e.Function)))
	case *InvokeFailed:
		l.Logger.Error("fx.Invoke failed",
			zap.Error(e.Err),
			zap.String("stack", e.Stacktrace),
			zap.String("function", fxreflect.FuncName(e.Function)))
	case *StartFailureError:
		l.Logger.Error("failed to start", zap.Error(e.Err))
	case *StopSignal:
		l.Logger.Info("received signal", zap.String("signal", strings.ToUpper(e.Signal.String())))
	case *StopError:
		l.Logger.Error("failed to stop cleanly", zap.Error(e.Err))
	case *StartRollbackError:
		l.Logger.Error("could not rollback cleanly", zap.Error(e.Err))
	case *StartError:
		l.Logger.Error("startup failed, rolling back", zap.Error(e.Err))
	case *Running:
		l.Logger.Info("running")
	}
}