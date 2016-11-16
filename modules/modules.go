// Copyright (c) 2016 Uber Technologies, Inc.
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

package modules

import (
	"go.uber.org/fx/service"

	"github.com/opentracing/opentracing-go"
	"github.com/uber-go/tally"
)

// A ModuleConfig holds configuration for a mobule
type ModuleConfig struct {
	Roles []string `yaml:"roles"`
}

// ModuleBase is an embeddable base for all UberFx modules
type ModuleBase struct {
	moduleType string
	name       string
	ctx        service.Context
	isRunning  bool
	roles      []string
	scope      tally.Scope
	tracer     opentracing.Tracer
}

// NewModuleBase configures a new ModuleBase
func NewModuleBase(
	moduleType string,
	name string,
	ctx service.Context,
	roles []string,
) *ModuleBase {
	return &ModuleBase{
		moduleType: moduleType,
		name:       name,
		ctx:        ctx,
		roles:      roles,
		scope:      ctx.Metrics().SubScope(name),
		tracer:     ctx.Tracer(),
	}
}

// Ctx returns module context
func (mb ModuleBase) Ctx() service.Context {
	return mb.ctx
}

// Roles returns the module's roles
func (mb ModuleBase) Roles() []string {
	return mb.roles
}

// Type returns the module's type
func (mb ModuleBase) Type() string {
	return mb.moduleType
}

// Name returns the module's name
func (mb ModuleBase) Name() string {
	return mb.name
}

// Tracer returns the module's service tracer
func (mb ModuleBase) Tracer() opentracing.Tracer {
	return mb.tracer
}
